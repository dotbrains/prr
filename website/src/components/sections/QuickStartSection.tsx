'use client';

import React, { useState } from 'react';
import { CodeBlock } from '@/components/CodeBlock';

export function QuickStartSection() {
  const [installMethod, setInstallMethod] = useState<'go' | 'brew' | 'release'>('go');

  const goExample = `go install github.com/dotbrains/prr@latest`;

  const brewExample = `brew tap dotbrains/tap
brew install --cask prr`;

  const releaseExample = `# macOS Apple Silicon
gh release download --repo dotbrains/prr \\
  --pattern 'prr_darwin_arm64.tar.gz' --dir /tmp
tar -xzf /tmp/prr_darwin_arm64.tar.gz -C /usr/local/bin`;

  const installExamples = { go: goExample, brew: brewExample, release: releaseExample };

  return (
    <section id="quick-start" className="py-12 sm:py-16 lg:py-20 bg-dark-slate overflow-hidden">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">
            Quick Start
          </h2>
          <p className="text-slate-gray text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            Install prr and run your first review in under a minute
          </p>
        </div>
        <div className="grid lg:grid-cols-2 gap-8 lg:gap-12 items-start">
          <div className="bg-dark-gray/50 rounded-xl p-6 sm:p-8 border border-prr-amber/20 min-w-0">
            <h3 className="text-xl sm:text-2xl font-bold text-cream mb-4 sm:mb-6">1. Install</h3>
            <div className="flex gap-2 sm:gap-3 mb-6">
              {[
                { key: 'go' as const, label: 'Go' },
                { key: 'brew' as const, label: 'Homebrew' },
                { key: 'release' as const, label: 'Release' },
              ].map((method) => (
                <button
                  key={method.key}
                  onClick={() => setInstallMethod(method.key)}
                  className={`flex-1 px-3 sm:px-4 py-2.5 rounded-lg text-sm font-semibold transition-all ${
                    installMethod === method.key
                      ? 'bg-gradient-to-r from-prr-amber to-prr-orange text-white shadow-lg shadow-prr-amber/30'
                      : 'bg-dark-slate text-slate-gray hover:text-cream hover:border-prr-amber/50 border border-prr-amber/30'
                  }`}
                >
                  {method.label}
                </button>
              ))}
            </div>
            <CodeBlock
              code={installExamples[installMethod]}
              language="bash"
            />
          </div>
          <div className="bg-dark-gray/50 rounded-xl p-6 sm:p-8 border border-prr-orange/20 min-w-0">
            <h3 className="text-xl sm:text-2xl font-bold text-cream mb-4 sm:mb-6">2. Review</h3>
            <CodeBlock
              code={`# Review the current branch's PR
prr

# Focus on security issues
prr 17509 --focus security

# Verify comments for accuracy
prr 17509 --verify

# Post review to GitHub
prr post

# Generate PR description
prr describe --update

# Ask a follow-up question
prr ask "Is this thread-safe?"

# Browse reviews in the web UI
prr serve --open`}
              language="bash"
            />
            <div className="mt-6 bg-prr-amber/10 border border-prr-amber/30 rounded-lg p-4 sm:p-5">
              <p className="text-cream text-sm leading-relaxed">
                <span className="text-prr-amber font-semibold">Tip:</span> Run <code className="bg-dark-slate/80 px-2 py-1 rounded text-prr-gold font-mono text-xs">prr config init</code> to scaffold a config file, then set your API key with <code className="bg-dark-slate/80 px-2 py-1 rounded text-prr-gold font-mono text-xs">export ANTHROPIC_API_KEY=sk-...</code>
              </p>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
