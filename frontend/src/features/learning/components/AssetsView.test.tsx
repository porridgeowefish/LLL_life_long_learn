import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { AssetsView } from './AssetsView';

vi.mock('@/features/learning/api/learningWorkspace',()=>({
 useAssets:()=>({data:[{key:'review',title:'复习稿'},{key:'practice',title:'练习题'}]}),
 useAsset:(_slug:string,key:string)=>({data:{meta:{key,title:key==='review'?'复习稿':'练习题',editRevision:1,currentVersionId:'aver_1'},content:key==='review'?'# 复习内容':'# 为什么成立？',contentKind:'markdown'}}),
 useSaveAsset:()=>({mutate:vi.fn(),isPending:false,isError:false}),
}));
vi.mock('./BodyAnnotations',()=>({BodyAnnotations:({content}:{content:string})=><p>{content}</p>}));

describe('AssetsView review and practice',()=>{
 it('shows review and practice without retired asset tabs',()=>{
  render(<AssetsView slug="topic"/>);
  expect(screen.getByRole('button',{name:'复习稿'})).toBeInTheDocument();
  expect(screen.getByRole('button',{name:'练习题'})).toBeInTheDocument();
  expect(screen.queryByRole('button',{name:/引入|正文|助教成果/})).not.toBeInTheDocument();
  expect(screen.getByText('# 复习内容')).toBeInTheDocument();
 });
 it('reads practice markdown and opens an editable draft',()=>{
  render(<AssetsView slug="topic"/>);
  fireEvent.click(screen.getByRole('button',{name:'练习题'}));
  expect(screen.getByRole('heading',{name:'为什么成立？'})).toBeInTheDocument();
  fireEvent.click(screen.getByRole('button',{name:'编辑'}));
  expect(screen.getByRole('textbox')).toHaveValue('# 为什么成立？');
 });
});
