// Package projects is the public boundary for project workspaces, folders, and
// the runtime project index. Filesystem details stay private to this module.
package projects

import (
	"github.com/xmz14/lll/backend-go/internal/modules/projects/internal/folderstore"
	"github.com/xmz14/lll/backend-go/internal/modules/projects/internal/projectindex"
	"github.com/xmz14/lll/backend-go/internal/modules/projects/internal/workspace"
)

type ZoneName = workspace.ZoneName
type ProjectType = workspace.ProjectType
type ProjectState = workspace.ProjectState
type ArtifactRef = workspace.ArtifactRef
type PredecessorFile = workspace.PredecessorFile
type ProjectMeta = workspace.ProjectMeta
type ProjectInput = workspace.ProjectInput
type SlugConflictError = workspace.SlugConflictError

const (
	ZoneIntro                 = workspace.ZoneIntro
	ZoneExplain               = workspace.ZoneExplain
	ZonePractice              = workspace.ZonePractice
	ZoneExtend                = workspace.ZoneExtend
	ZoneSummary               = workspace.ZoneSummary
	ProjectTypeDisciplineMap  = workspace.ProjectTypeDisciplineMap
	ProjectTypeSystemLearning = workspace.ProjectTypeSystemLearning
)

var AllZones = workspace.AllZones
var NormalizeProjectType = workspace.NormalizeProjectType
var ValidateProjectType = workspace.ValidateProjectType
var ValidateSlug = workspace.ValidateSlug
var ValidateZoneName = workspace.ValidateZoneName
var ProjectExists = workspace.ProjectExists
var DeleteProject = workspace.DeleteProject
var CreateProjectSkeleton = workspace.CreateProjectSkeleton
var CreateProjectSkeletonWithInput = workspace.CreateProjectSkeletonWithInput
var ReadProjectState = workspace.ReadProjectState
var WriteProjectState = workspace.WriteProjectState
var IndexAll = workspace.IndexAll
var ResolvePredecessorFiles = workspace.ResolvePredecessorFiles
var SafeWriteArtifact = workspace.SafeWriteArtifact
var SafeWriteSummary = workspace.SafeWriteSummary
var ReadArtifact = workspace.ReadArtifact
var WriteRunFile = workspace.WriteRunFile
var IsSlugConflict = workspace.IsSlugConflict
var FindProjectBySlug = workspace.FindProjectBySlug
var EnsureProjectsRoot = workspace.EnsureProjectsRoot
var ProjectRootForSlug = workspace.ProjectRootForSlug
var ProjectsRootForTest = workspace.ProjectsRootForTest
var SetProjectsRootForTest = workspace.SetProjectsRootForTest
var AtomicWriteFile = workspace.AtomicWriteFile
var BackupCorruptFile = workspace.BackupCorruptFile
var Slugify = workspace.Slugify

type Folder = folderstore.Folder
type MapFolderSpec = folderstore.MapFolderSpec
type Layout = folderstore.Layout
type FolderLayout = folderstore.Layout
type FolderStore = folderstore.Store

func NewFolderStore() (*FolderStore, error) { return folderstore.New() }

type Index = projectindex.Cache

func NewIndex() *Index { return projectindex.New() }
