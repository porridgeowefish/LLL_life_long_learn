// create-project-modal.js — rich form for creating a learning project.
// Replaces the old prompt() single-field UX per LEARNING_PROJECT_STRUCTURE.md
// (background collection: why / current / target / standard).

import { createProjectRich } from './api.js';

const LEVELS = ['未接触', '了解概念', '动手做过', '能独立完成'];

/**
 * Mount the create-project modal into the page (idempotent).
 * Returns a function that opens the modal.
 */
export function mountCreateProjectModal({ onSuccess, mount = document.body } = {}) {
  // Inject DOM once.
  if (document.getElementById('create-modal-overlay')) {
    return () => document.getElementById('create-modal-overlay').classList.add('open');
  }

  const overlay = document.createElement('div');
  overlay.className = 'modal-overlay';
  overlay.id = 'create-modal-overlay';
  overlay.innerHTML = `
    <div class="modal" role="dialog" aria-labelledby="create-modal-title">
      <div class="modal-head">
        <h3 id="create-modal-title">新建学习项目</h3>
        <button class="btn-tiny" id="create-modal-close" aria-label="关闭">关闭</button>
      </div>
      <form id="create-modal-form" class="modal-body">
        <div class="field">
          <label for="f-title">项目标题 <span style="color:var(--pink)">*</span></label>
          <input id="f-title" name="title" type="text" required placeholder="例如：Rust 所有权模型" autocomplete="off">
          <div class="hint">用作项目目录名（自动 slugify），后续可改显示标题。</div>
        </div>

        <div class="field">
          <label for="f-why">为什么学这个？ <span style="color:var(--pink)">*</span></label>
          <textarea id="f-why" name="why" required placeholder="一句话说明动机。例如：「理解所有权是写好 Rust 的前提，避免在 borrow checker 前束手无策。」"></textarea>
          <div class="hint">写入 project.md 的 "Why this topic matters" 章节，会成为智能体调用时的上下文。</div>
        </div>

        <div class="field">
          <label>当前水平</label>
          <div class="field-radio" id="f-current">
            ${LEVELS.map((lvl, i) => `<label><input type="radio" name="current" value="${lvl}" ${i===0?'checked':''}>${lvl}</label>`).join('')}
          </div>
        </div>

        <div class="field">
          <label>目标水平</label>
          <div class="field-radio" id="f-target">
            ${LEVELS.map((lvl, i) => `<label><input type="radio" name="target" value="${lvl}" ${i===LEVELS.length-1?'checked':''}>${lvl}</label>`).join('')}
          </div>
        </div>

        <div class="field">
          <label for="f-standard">完成标准</label>
          <textarea id="f-standard" name="standard" placeholder="怎样的成果算"学完"？例如：能独立解释三条所有权规则并写出 5 行无编译错误的借用代码。"></textarea>
          <div class="hint">可选；用于判断何时进入下一阶段。</div>
        </div>
      </form>
      <div class="modal-foot">
        <button class="btn btn-outline" id="create-modal-cancel">取消</button>
        <button class="btn btn-primary" id="create-modal-submit">创建项目</button>
      </div>
    </div>
  `;
  mount.appendChild(overlay);

  const close = () => overlay.classList.remove('open');
  overlay.addEventListener('click', e => { if (e.target === overlay) close(); });
  document.getElementById('create-modal-close').addEventListener('click', close);
  document.getElementById('create-modal-cancel').addEventListener('click', close);

  const submit = async () => {
    const form = document.getElementById('create-modal-form');
    const fd = new FormData(form);
    const payload = {
      title: (fd.get('title') || '').trim(),
      why: (fd.get('why') || '').trim(),
      current: fd.get('current') || LEVELS[0],
      target: fd.get('target') || LEVELS[LEVELS.length - 1],
      standard: (fd.get('standard') || '').trim(),
    };
    if (!payload.title || !payload.why) {
      alert('请填写项目标题和"为什么学这个"');
      return;
    }
    const btn = document.getElementById('create-modal-submit');
    btn.disabled = true;
    btn.textContent = '创建中…';
    try {
      const { project } = await createProjectRich(payload);
      close();
      // reset form for next open
      form.reset();
      if (onSuccess) onSuccess(project);
    } catch (e) {
      alert('创建失败: ' + e.message);
    } finally {
      btn.disabled = false;
      btn.textContent = '创建项目';
    }
  };
  document.getElementById('create-modal-submit').addEventListener('click', submit);

  return () => overlay.classList.add('open');
}
