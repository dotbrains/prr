'use client';

import { GitPullRequest, GitBranch, Send, Eye, Shield, Clock } from 'lucide-react';

export function UseCasesSection() {
  const useCases = [
    {
      icon: <GitPullRequest className="w-6 h-6" />,
      title: 'PR Reviews',
      description: 'Get a thorough AI review on any GitHub PR by number or URL. Catches bugs, race conditions, and security issues.',
    },
    {
      icon: <GitBranch className="w-6 h-6" />,
      title: 'Pre-PR Reviews',
      description: 'Review your branch locally before opening a PR. Fix issues before anyone sees them.',
    },
    {
      icon: <Send className="w-6 h-6" />,
      title: 'Review → Post Workflow',
      description: 'Review a PR, inspect the output, then post directly to GitHub with prr post. No copy-paste needed.',
    },
    {
      icon: <Eye className="w-6 h-6" />,
      title: 'Self-Review',
      description: 'Use prr on your own PRs before requesting review. Generate a description with prr describe. Reduce review rounds.',
    },
    {
      icon: <Shield className="w-6 h-6" />,
      title: 'Focused Security Audit',
      description: 'Run prr --focus security to zero in on injection, auth bypass, and crypto issues. Suppresses noise from unrelated nits.',
    },
    {
      icon: <Clock className="w-6 h-6" />,
      title: 'Incremental Reviews',
      description: 'Use --since to only review changes since the last run. Track progress between iterations with prr diff.',
    },
  ];

  return (
    <section id="use-cases" className="py-12 sm:py-16 lg:py-20 bg-dark-slate">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">
            Use Cases
          </h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            prr adapts to how your team reviews code
          </p>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6 lg:gap-8">
          {useCases.map((useCase, index) => (
            <div
              key={index}
              className="bg-dark-gray/50 border border-prr-amber/20 rounded-xl p-5 sm:p-6 hover:border-prr-orange/40 transition-all"
            >
              <div className="w-10 h-10 sm:w-12 sm:h-12 bg-gradient-to-br from-prr-amber to-prr-orange rounded-lg flex items-center justify-center text-white mb-3 sm:mb-4">
                {useCase.icon}
              </div>
              <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2">{useCase.title}</h3>
              <p className="text-cream/60 text-sm sm:text-base leading-relaxed">{useCase.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
