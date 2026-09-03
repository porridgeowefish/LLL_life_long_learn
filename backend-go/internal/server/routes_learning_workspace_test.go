package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/xmz14/lll/backend-go/internal/conversationstore"
	"github.com/xmz14/lll/backend-go/internal/teachergateway"
	"github.com/xmz14/lll/backend-go/internal/teacherservice"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

type textTeacherGateway struct{}

func (textTeacherGateway) Stream(_ context.Context, _ teachergateway.Request, emit func(teachergateway.Event)) error {
	emit(teachergateway.Event{Type: "text-delta", Delta: "我们先从定义开始。"})
	emit(teachergateway.Event{Type: "response-completed"})
	return nil
}

type blockingTeacherGateway struct {
	started chan struct{}
	release chan struct{}
}

func (g blockingTeacherGateway) Stream(_ context.Context, _ teachergateway.Request, emit func(teachergateway.Event)) error {
	close(g.started)
	emit(teachergateway.Event{Type: "text-delta", Delta: "先给你一个思考方向。"})
	<-g.release
	emit(teachergateway.Event{Type: "text-delta", Delta: "现在继续完成回答。"})
	emit(teachergateway.Event{Type: "response-completed"})
	return nil
}

func learningWorkspaceServer(t *testing.T) *Server {
	t.Helper()
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	t.Cleanup(func() { workspace.SetProjectsRootForTest("") })
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	return &Server{teacher: teacherservice.New(textTeacherGateway{}), activeTeacher: map[string]*activeTeacherRun{}, activeByProject: map[string]*activeTeacherRun{}, migrationReady: true}
}

func TestTeacherTurnStreamsAndPersists(t *testing.T) {
	server := learningWorkspaceServer(t)
	req := httptest.NewRequest(http.MethodPost, "/api/projects/topic/conversation/turns", strings.NewReader(`{"operationId":"op_test","content":"什么是闭包？"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "text-delta") || !strings.Contains(response.Body.String(), "我们先从定义开始") {
		t.Fatalf("unexpected stream: %d %s", response.Code, response.Body.String())
	}
	replay := httptest.NewRecorder()
	server.Handler().ServeHTTP(replay, httptest.NewRequest(http.MethodPost, "/api/projects/topic/conversation/turns", strings.NewReader(`{"operationId":"op_test","content":"什么是闭包？"}`)))
	if replay.Code != http.StatusOK || !strings.Contains(replay.Body.String(), "我们先从定义开始") {
		t.Fatalf("idempotent replay failed: %d %s", replay.Code, replay.Body.String())
	}

	read := httptest.NewRecorder()
	server.Handler().ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/projects/topic/conversation?limit=500", nil))
	var projection struct {
		Messages []struct {
			Role string `json:"role"`
		} `json:"messages"`
	}
	if err := json.Unmarshal(read.Body.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	if len(projection.Messages) != 3 || projection.Messages[0].Role != "teacher" || projection.Messages[1].Role != "learner" || projection.Messages[2].Role != "teacher" {
		t.Fatalf("conversation not durable: %s", read.Body.String())
	}
}

func TestTeacherRunSurvivesDisconnectedSubscriber(t *testing.T) {
	server := learningWorkspaceServer(t)
	gateway := blockingTeacherGateway{started: make(chan struct{}), release: make(chan struct{})}
	server.teacher = teacherservice.New(gateway)
	requestContext, disconnect := context.WithCancel(context.Background())
	req := httptest.NewRequest(http.MethodPost, "/api/projects/topic/conversation/turns", strings.NewReader(`{"operationId":"op_disconnect","content":"刷新也要继续回答"}`)).WithContext(requestContext)
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	finished := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(response, req)
		close(finished)
	}()
	select {
	case <-gateway.started:
	case <-time.After(time.Second):
		t.Fatal("teacher did not start")
	}
	disconnect()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("disconnected subscriber did not return")
	}

	resume := httptest.NewRecorder()
	activeContext, cancelActive := context.WithCancel(context.Background())
	activeRequest := httptest.NewRequest(http.MethodGet, "/api/projects/topic/conversation/responses/active", nil).WithContext(activeContext)
	activeFinished := make(chan struct{})
	go func() {
		server.Handler().ServeHTTP(resume, activeRequest)
		close(activeFinished)
	}()
	time.Sleep(20 * time.Millisecond)
	cancelActive()
	select {
	case <-activeFinished:
	case <-time.After(time.Second):
		t.Fatal("active response subscriber did not return")
	}
	if resume.Code != http.StatusOK || !strings.Contains(resume.Body.String(), "先给你一个思考方向") {
		t.Fatalf("active response was not resumable: %d %s", resume.Code, resume.Body.String())
	}
	close(gateway.release)
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		read := httptest.NewRecorder()
		server.Handler().ServeHTTP(read, httptest.NewRequest(http.MethodGet, "/api/projects/topic/conversation?limit=500", nil))
		if strings.Contains(read.Body.String(), "现在继续完成回答") {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("teacher did not finish after subscriber disconnected")
}

func TestConversationCreatesOneTeacherGreeting(t *testing.T) {
	server := learningWorkspaceServer(t)
	for attempt := 0; attempt < 2; attempt++ {
		response := httptest.NewRecorder()
		server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/projects/topic/conversation?limit=500", nil))
		if response.Code != http.StatusOK {
			t.Fatalf("conversation read failed: %d %s", response.Code, response.Body.String())
		}
		var projection struct {
			Messages []struct {
				Role   string `json:"role"`
				Blocks []struct {
					Source string `json:"source"`
				} `json:"blocks"`
			} `json:"messages"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &projection); err != nil {
			t.Fatal(err)
		}
		if len(projection.Messages) != 1 || projection.Messages[0].Role != "teacher" || len(projection.Messages[0].Blocks) != 1 || !strings.Contains(projection.Messages[0].Blocks[0].Source, "最想先弄懂什么") {
			t.Fatalf("unexpected greeting projection: %s", response.Body.String())
		}
	}
}

func TestConversationTailEndpointReturnsOnlyRecentMessages(t *testing.T) {
	server := learningWorkspaceServer(t)
	store, err := conversationstore.New("topic")
	if err != nil {
		t.Fatal(err)
	}
	if err := ensureTeacherGreeting(store, "topic"); err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 45; index++ {
		if _, _, err := store.AppendMessage("learner", "completed", fmt.Sprintf("tail-%d", index), []conversationstore.Block{{Type: "markdown", Source: fmt.Sprintf("question-%d", index)}}); err != nil {
			t.Fatal(err)
		}
	}
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/projects/topic/conversation?limit=10&beforeSeq=0", nil))
	var projection conversationstore.Projection
	if err := json.Unmarshal(response.Body.Bytes(), &projection); err != nil {
		t.Fatal(err)
	}
	if response.Code != http.StatusOK || len(projection.Messages) != 10 || projection.TotalMessages != 46 || !projection.HasPrevious {
		t.Fatalf("unexpected recent projection: %d %s", response.Code, response.Body.String())
	}
	if got := projection.Messages[0].Blocks[0].Source; got != "question-35" {
		t.Fatalf("recent endpoint did not start at the expected tail: %q", got)
	}
}

func TestAssetOptimisticRevisionConflict(t *testing.T) {
	server := learningWorkspaceServer(t)
	get := httptest.NewRecorder()
	server.Handler().ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/api/projects/topic/assets/body", nil))
	var asset struct {
		Meta struct {
			EditRevision uint64 `json:"editRevision"`
		} `json:"meta"`
	}
	_ = json.Unmarshal(get.Body.Bytes(), &asset)
	// The stale second request deliberately reuses the original edit revision.
	body := `{"baseEditRevision":` + strconv.FormatUint(asset.Meta.EditRevision, 10) + `,"content":"正文"}`
	first := httptest.NewRecorder()
	server.Handler().ServeHTTP(first, httptest.NewRequest(http.MethodPut, "/api/projects/topic/assets/body", strings.NewReader(body)))
	if first.Code != http.StatusOK {
		t.Fatalf("first save failed: %d %s", first.Code, first.Body.String())
	}
	stale := httptest.NewRecorder()
	server.Handler().ServeHTTP(stale, httptest.NewRequest(http.MethodPut, "/api/projects/topic/assets/body", strings.NewReader(body)))
	if stale.Code != http.StatusConflict || !strings.Contains(stale.Body.String(), "asset_edit_conflict") {
		t.Fatalf("stale edit was not rejected: %d %s", stale.Code, stale.Body.String())
	}
}

func TestArchiveUploadStaysOpaqueAndDoesNotCreateParseTask(t *testing.T) {
	server := learningWorkspaceServer(t)
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	_ = writer.WriteField("operationId", "op_archive")
	_ = writer.WriteField("parseApproved", "true")
	_ = writer.WriteField("cloudDisclosureAccepted", "true")
	part, err := writer.CreateFormFile("file", "materials.zip")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("PK\x03\x04not-a-real-archive")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/projects/topic/sources", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("upload failed: %d %s", response.Code, response.Body.String())
	}
	var payload struct {
		Source struct {
			Status string `json:"status"`
		} `json:"source"`
		Task             *json.RawMessage `json:"task"`
		ParseDisposition string           `json:"parseDisposition"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Source.Status != "opaque" || payload.Task != nil || payload.ParseDisposition != "opaque" {
		t.Fatalf("archive must remain opaque without a task: %s", response.Body.String())
	}
}
