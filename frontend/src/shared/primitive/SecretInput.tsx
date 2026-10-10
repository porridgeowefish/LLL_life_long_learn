import { useState } from 'react';
import s from './SecretInput.module.css';
export function SecretInput({value,onChange}:{value:string;onChange:(value:string)=>void}) {
 const [visible,setVisible]=useState(false);
 return <span className={s.control}><input aria-label="API Key" type={visible?'text':'password'} value={value} onChange={event=>onChange(event.target.value)} autoComplete="off" spellCheck={false}/><button type="button" aria-label={visible?'隐藏 API Key':'显示 API Key'} aria-pressed={visible} onClick={()=>setVisible(current=>!current)}>{visible?'隐藏':'显示'}</button></span>;
}
