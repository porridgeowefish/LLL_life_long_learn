import {ApiError} from '@/shared/client';

const messages: Record<string,string> = {
 'provider-unavailable':'所选 API 连接不可用，请检查 Key 和连接配置。',
 'model-unavailable':'未配置有效模型，请选择或填写模型名称。',
 'input-budget-exhausted':'本次资料与对话超过模型上下文预算，请减少资料或调整连接的上下文容量。',
 'generation-active':'此内容正在生成，请查看当前任务进度。',
 'generation-queue-full':'生成队列已满，请稍后重试。',
 'generation-project-type':'此页面不支持这类生成，请从对应学习单元操作。',
 'source-not-readable':'所选资料尚未解析完成，请等待或取消选择该资料。',
 'source-unavailable':'所选资料已不可用，请重新选择。',
 'source-selection-required':'请先选择可用的参考资料。',
 'source-structure-incomplete':'参考资料的章节结构不完整，请补充资料后重试。',
 'prompt-resource-unavailable':'教学生成资源缺失或版本不一致，请修复或更新安装包。',
 'migration-blocked':'旧任务尚未完成兼容迁移，请先关闭旧版程序后重新启动。',
 'generation-unavailable':'生成服务未就绪，请重新启动应用。',
 'provider-failed':'模型服务调用失败，请检查连接、套餐权限与额度。',
 'output-limit':'模型已达到单次输出上限，正文可能被截断。请减少题数或分批总结后重试。',
 'empty-output':'模型没有返回正文，请重试或检查模型的思考与输出配置。',
 'invalid-practice-output':'练习题未完整满足题数、答案或解析要求，请减少题数后重试。',
 'invalid-map-output':'地图正文与主题目录不一致，未替换旧地图，请重试。',
 'invalid-review-output':'复习稿缺少完整章节，未替换旧稿，请重试。',
 'edit-conflict':'生成期间原稿已修改，旧稿已保留，请重新生成。',
 'request-interrupted':'生成已中断，可重新发起。',
 'operation-conflict':'本次提交内容已变化，请关闭弹窗后重新生成。',
};
export function describeGenerationError(error: unknown): string {
 if(typeof error==='string')return Object.prototype.hasOwnProperty.call(messages,error)?messages[error]:'生成未完成，请重试或检查连接。';
 if(error instanceof ApiError){
  const body=error.body as {error?: {code?: unknown}}|null;
  const code=typeof body?.error==='object'&&typeof body.error?.code==='string'?body.error.code:'';
  if(Object.prototype.hasOwnProperty.call(messages,code))return messages[code];
  if(error.status===0)return '无法连接本地服务，请确认应用仍在运行后重试。';
  return `提交未完成（HTTP ${error.status}），请重试。`;
 }
 return '提交未完成，请重试。';
}
