import type { PracticeQuestion } from './PracticeReader';

/** Answer key of one objective question, normalised to option ids or
 * 'true'/'false'. null means the question cannot be scored automatically
 * (historic short-answer/code questions or an unreadable key). */
export function expectedAnswer(question: PracticeQuestion): string[] | null {
  const lines = question.answer.split(/\r?\n/);
  const heading = lines.findIndex(line => /^#{1,6}\s*参考答案\s*$/.test(line.trim()));
  const first = lines.slice(heading + 1).map(line => line.trim()).find(line => line && !/^#{1,6}\s/.test(line));
  if (!first) return null;
  const key = first.replace(/[*_`]/g, '').replace(/^(?:正确)?答案\s*[：:]\s*/, '').trim();
  if (question.kind === 'true-false') {
    if (/^(?:不正确|不对|错误|错|false|f|×|✗)(?:[。.]|$)/i.test(key)) return ['false'];
    if (/^(?:正确|对|true|t|√|✓)(?:[。.]|$)/i.test(key)) return ['true'];
    return null;
  }
  const ids = new Set((question.options ?? []).map(option => option.id));
  if (!ids.size) return null;
  const letters = [...new Set(key.match(/(?<![A-Za-z])[A-Z](?![A-Za-z])/g) ?? [])].filter(id => ids.has(id)).sort();
  if (!letters.length) return null;
  if (question.kind === 'single-choice' && letters.length !== 1) return null;
  return letters;
}

export function isCorrect(selected: string[] | undefined, expected: string[]) {
  const chosen = [...new Set(selected ?? [])].sort();
  return chosen.length === expected.length && chosen.every((id, index) => id === expected[index]);
}

export interface GradedTest { answers: Record<string, string[]>; items: Record<string, { correct: boolean; expected: string[] }>; score: number; total: number }

/** Deterministic scoring of every gradable question; keys are 1-based question numbers. */
export function gradeTest(questions: PracticeQuestion[], selections: Record<string, string[]>): GradedTest | null {
  const items: GradedTest['items'] = {};
  const answers: GradedTest['answers'] = {};
  questions.forEach((question, index) => {
    const expected = expectedAnswer(question);
    if (!expected) return;
    const id = String(index + 1);
    const chosen = selections[id] ?? [];
    answers[id] = chosen;
    items[id] = { correct: isCorrect(chosen, expected), expected };
  });
  const total = Object.keys(items).length;
  if (!total) return null;
  return { answers, items, score: Object.values(items).filter(item => item.correct).length, total };
}

export function choiceLabel(question: PracticeQuestion, ids: string[]) {
  if (question.kind === 'true-false') return ids.map(id => id === 'true' ? '正确' : '错误').join('、');
  return ids.join('、');
}
