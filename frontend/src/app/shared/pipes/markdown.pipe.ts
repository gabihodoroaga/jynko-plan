import { Pipe, PipeTransform } from '@angular/core';

@Pipe({
  name: 'markdown',
  standalone: true,
})
export class MarkdownPipe implements PipeTransform {
  transform(value: string | null | undefined): string {
    if (!value) return '';

    const escaped = value
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');

    const lines = escaped.split('\n');
    const out: string[] = [];
    let inCode = false;

    for (const rawLine of lines) {
      const line = rawLine;
      if (line.trim().startsWith('```')) {
        if (!inCode) {
          inCode = true;
          out.push(
            '<pre class="my-2 p-3 rounded-md bg-slate-100 border border-slate-200 font-mono text-xs text-slate-800 overflow-x-auto"><code>'
          );
        } else {
          inCode = false;
          out.push('</code></pre>');
        }
        continue;
      }

      if (inCode) {
        out.push(line + '\n');
        continue;
      }

      if (line.startsWith('### ')) {
        out.push(
          `<h3 class="text-sm font-semibold text-slate-900 mt-4 mb-1">${this.inlineFormat(line.slice(4))}</h3>`
        );
      } else if (line.startsWith('## ')) {
        out.push(
          `<h2 class="text-base font-semibold text-slate-900 mt-5 mb-1.5 pb-1 border-b border-slate-200">${this.inlineFormat(line.slice(3))}</h2>`
        );
      } else if (line.startsWith('# ')) {
        out.push(
          `<h1 class="text-lg font-bold text-slate-900 mt-2 mb-2">${this.inlineFormat(line.slice(2))}</h1>`
        );
      } else if (line.startsWith('- [x] ') || line.startsWith('- [X] ')) {
        out.push(
          `<div class="flex items-center gap-2 my-1 text-xs text-slate-500 line-through"><span class="inline-flex items-center justify-center w-3.5 h-3.5 rounded bg-emerald-100 text-emerald-700 border border-emerald-300 text-[10px]">✓</span><span>${this.inlineFormat(line.slice(6))}</span></div>`
        );
      } else if (line.startsWith('- [ ] ')) {
        out.push(
          `<div class="flex items-center gap-2 my-1 text-xs text-slate-700"><span class="inline-block w-3.5 h-3.5 rounded border border-slate-300 bg-white"></span><span>${this.inlineFormat(line.slice(6))}</span></div>`
        );
      } else if (line.startsWith('- ')) {
        out.push(
          `<div class="flex items-start gap-2 my-1 text-xs text-slate-700"><span class="text-slate-400 mt-0.5">•</span><span>${this.inlineFormat(line.slice(2))}</span></div>`
        );
      } else if (line.startsWith('&gt; ')) {
        out.push(
          `<blockquote class="my-2 pl-3 border-l-2 border-sky-600 text-xs italic text-slate-600 bg-slate-50 py-1 pr-2 rounded-r">${this.inlineFormat(line.slice(5))}</blockquote>`
        );
      } else if (line.trim() === '---') {
        out.push('<hr class="my-3 border-slate-200" />');
      } else if (line.trim() === '') {
        out.push('<div class="h-2"></div>');
      } else {
        out.push(
          `<p class="text-xs leading-relaxed text-slate-700 my-1">${this.inlineFormat(line)}</p>`
        );
      }
    }

    if (inCode) {
      out.push('</code></pre>');
    }

    return out.join('');
  }

  private inlineFormat(text: string): string {
    return text
      .replace(
        /`([^`]+)`/g,
        '<code class="px-1 py-0.5 rounded bg-slate-100 border border-slate-200 font-mono text-[11px] text-slate-800">$1</code>'
      )
      .replace(/\*\*([^*]+)\*\*/g, '<strong class="font-semibold text-slate-900">$1</strong>')
      // Images: ![alt](url) where url is http(s):// or /
      .replace(
        /!\[([^\]]*)\]\((https?:\/\/[^\s)]+|\/[^\s)]+)\)/g,
        '<img src="$2" alt="$1" class="max-w-full max-h-96 rounded-lg border border-slate-200 shadow-sm my-2 object-contain bg-white cursor-pointer hover:border-sky-300 transition-colors" loading="lazy" />'
      )
      // Links: [text](url) where url is http(s):// or /
      .replace(
        /\[([^\]]+)\]\((https?:\/\/[^\s)]+|\/[^\s)]+)\)/g,
        '<a href="$2" target="_blank" rel="noopener noreferrer" class="text-sky-600 hover:text-sky-700 underline font-medium inline-flex items-center gap-1"><span>$1</span><svg class="w-3 h-3 inline-block shrink-0 opacity-70" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M4.25 5.5a.75.75 0 00-.75.75v8.5c0 .414.336.75.75.75h8.5a.75.75 0 00.75-.75v-4a.75.75 0 011.5 0v4A2.25 2.25 0 0112.75 17h-8.5A2.25 2.25 0 012 14.75v-8.5A2.25 2.25 0 014.25 4h4a.75.75 0 010 1.5h-4z" clip-rule="evenodd" /><path fill-rule="evenodd" d="M6.194 12.753a.75.75 0 001.06 1.06l7.22-7.22v2.657a.75.75 0 001.5 0v-4.5a.75.75 0 00-.75-.75h-4.5a.75.75 0 000 1.5h2.657l-7.187 7.253z" clip-rule="evenodd" /></svg></a>'
      );
  }
}
