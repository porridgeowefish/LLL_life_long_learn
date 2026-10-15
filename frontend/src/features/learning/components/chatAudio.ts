// WAV is the shared input format of the system recognizers. No audio leaves the app.
export function encodeWave(samples: Float32Array, sampleRate: number): ArrayBuffer {
  if (!Number.isFinite(sampleRate) || sampleRate < 8000 || !samples.length || samples.length / sampleRate > 60) throw new Error('音频必须在60秒以内');
  const count = Math.floor(samples.length * 16000 / sampleRate);
  const output = new ArrayBuffer(44 + count * 2);
  const view = new DataView(output);
  const text = (offset: number, value: string) => [...value].forEach((c, i) => view.setUint8(offset + i, c.charCodeAt(0)));
  text(0, 'RIFF'); view.setUint32(4, 36 + count * 2, true); text(8, 'WAVE'); text(12, 'fmt ');
  view.setUint32(16, 16, true); view.setUint16(20, 1, true); view.setUint16(22, 1, true);
  view.setUint32(24, 16000, true); view.setUint32(28, 32000, true); view.setUint16(32, 2, true); view.setUint16(34, 16, true);
  text(36, 'data'); view.setUint32(40, count * 2, true);
  for (let i = 0; i < count; i++) {
    const start = Math.floor(i * sampleRate / 16000);
    const end = Math.max(start + 1, Math.floor((i + 1) * sampleRate / 16000));
    let value = 0;
    for (let j = start; j < Math.min(end, samples.length); j++) value += samples[j];
    value = Math.max(-1, Math.min(1, value / (Math.min(end, samples.length) - start)));
    view.setInt16(44 + i * 2, Math.round(value * (value < 0 ? 32768 : 32767)), true);
  }
  return output;
}

export async function audioFileToWave(file: File): Promise<File> {
  const context = new AudioContext();
  try {
    const buffer = await context.decodeAudioData(await file.arrayBuffer());
    if (buffer.duration > 60) throw new Error('音频必须在60秒以内');
    const mono = new Float32Array(buffer.length);
    for (let channel = 0; channel < buffer.numberOfChannels; channel++) {
      const values = buffer.getChannelData(channel);
      for (let i = 0; i < mono.length; i++) mono[i] += values[i] / buffer.numberOfChannels;
    }
    return new File([encodeWave(mono, buffer.sampleRate)], file.name.replace(/\.[^.]+$/, '') + '.wav', { type: 'audio/wav' });
  } catch (error) {
    if (error instanceof Error && error.message.includes('60')) throw error;
    throw new Error('无法解码此音频，请使用WAV、MP3、M4A或录音按钮');
  } finally { await context.close(); }
}
