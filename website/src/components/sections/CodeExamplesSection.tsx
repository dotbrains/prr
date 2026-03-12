'use client';

import React, { useState } from 'react';
import { CodeBlock } from '@/components/CodeBlock';

export function CodeExamplesSection() {
  const [activeTab, setActiveTab] = useState<'pr' | 'local' | 'config'>('pr');

  const examples = {
    pr: `# Review the current branch's PR
$ prr
→ PR #17509: Fix user authentication race condition
→ agent:  claude (claude-opus-4-20250514)
→ files:  12 (3 filtered)
→ Reviewing...

✓ Review complete.
→ 2 critical, 5 suggestions, 3 nits, 1 praise
→ Output: reviews/pr-17509-20250311-143000/

# Review a specific PR
$ prr 17509 --agent gpt

# Review any PR by URL (no cloning needed)
$ prr https://github.com/owner/repo/pull/123`,
    local: `# Review current branch against main
$ prr --base main
→ Local review: main → feature/auth
→ repo:  .
→ files:  8 (2 filtered)
→ agent:  claude (opus)
→ Reviewing...

✓ Review complete.
→ 1 critical, 3 suggestions, 2 nits
→ Output: reviews/review-main-vs-feature-auth-20250311/

# Review a specific repo and branch
$ prr --repo ../other-project --base develop --head feature/api`,
    config: `# ~/.config/prr/config.yaml
default_agent: claude-cli

agents:
  claude-cli:
    provider: claude-cli
    model: opus

  claude-api:
    provider: anthropic
    model: claude-opus-4-20250514
    api_key_env: ANTHROPIC_API_KEY
    max_tokens: 8192

  gpt-api:
    provider: openai
    model: gpt-4o
    api_key_env: OPENAI_API_KEY
    max_tokens: 8192

review:
  max_diff_lines: 10000
  ignore_patterns:
    - "*.lock"
    - "go.sum"
    - "vendor/**"
    - "node_modules/**"`,
  };

  const tabs = [
    { key: 'pr' as const, label: 'PR Mode', language: 'bash' },
    { key: 'local' as const, label: 'Local Mode', language: 'bash' },
    { key: 'config' as const, label: 'Configuration', language: 'yaml' },
  ];

  return (
    <section id="code-examples" className="py-12 sm:py-16 lg:py-20 bg-dark-gray/50">
      <div className="max-w-6xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">
            Code Examples
          </h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            See prr in action — PR reviews, local diffs, and configuration
          </p>
        </div>
        <div className="bg-dark-slate border border-prr-amber/30 rounded-xl overflow-hidden">
          <div className="flex border-b border-prr-amber/30 overflow-x-auto">
            {tabs.map((tab) => (
              <button
                key={tab.key}
                onClick={() => setActiveTab(tab.key)}
                className={`flex-1 px-3 sm:px-6 py-3 sm:py-4 text-xs sm:text-sm font-semibold transition-colors whitespace-nowrap ${
                  activeTab === tab.key
                    ? 'bg-dark-gray/50 text-prr-amber border-b-2 border-prr-amber'
                    : 'text-cream/70 hover:text-cream hover:bg-dark-gray/30'
                }`}
              >
                {tab.label}
              </button>
            ))}
          </div>
          <div className="p-4 sm:p-6 overflow-x-auto">
            <CodeBlock
              code={examples[activeTab]}
              language={tabs.find((t) => t.key === activeTab)?.language}
            />
          </div>
        </div>
      </div>
    </section>
  );
}
