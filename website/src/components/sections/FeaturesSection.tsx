'use client';

import { Bot, GitPullRequest, GitBranch, Users, FileText, Zap } from 'lucide-react';

export function FeaturesSection() {
  const features = [
    {
      icon: <Bot className="w-6 h-6" />,
      title: 'Claude & GPT Agents',
      description: 'Choose from Claude Opus, GPT-4o, or bring your own. CLI and API providers supported out of the box.',
    },
    {
      icon: <GitPullRequest className="w-6 h-6" />,
      title: 'GitHub PR Reviews',
      description: 'Pass a PR number or URL — prr fetches the diff via gh, sends it to AI, and writes structured review comments.',
    },
    {
      icon: <GitBranch className="w-6 h-6" />,
      title: 'Local Branch Diffs',
      description: 'No PR needed. Diff any two branches in a local git repo and get a full AI code review.',
    },
    {
      icon: <Users className="w-6 h-6" />,
      title: 'Human-Like Comments',
      description: 'Prompt-engineered to write like a senior engineer — direct, specific, no AI-speak. Banned phrases list enforced.',
    },
    {
      icon: <FileText className="w-6 h-6" />,
      title: 'Structured Output',
      description: 'One markdown file per reviewed source file. Organized by line number with severity levels. Ready to paste into GitHub.',
    },
    {
      icon: <Zap className="w-6 h-6" />,
      title: 'Multi-Agent Reviews',
      description: 'Run all agents in parallel with --all. Compare Claude vs GPT reviews side-by-side.',
    },
  ];

  return (
    <section id="features" className="py-12 sm:py-16 lg:py-20 bg-dark-slate">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-10 sm:mb-16">
          <h2 className="text-3xl sm:text-4xl lg:text-5xl font-bold text-cream mb-3 sm:mb-4">
            Built for Real Engineering Workflows
          </h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl max-w-3xl mx-auto">
            prr fits into how you already work — local-first, provider-agnostic, zero bot spam
          </p>
        </div>
        <div className="grid sm:grid-cols-2 lg:grid-cols-3 gap-4 sm:gap-6 lg:gap-8">
          {features.map((feature, index) => (
            <div
              key={index}
              className="group bg-dark-gray/50 border border-prr-purple/20 hover:border-prr-indigo/40 rounded-xl p-5 sm:p-6 transition-all hover:shadow-lg hover:shadow-prr-purple/10"
            >
              <div className="w-10 h-10 sm:w-12 sm:h-12 bg-gradient-to-br from-prr-purple to-prr-indigo rounded-lg flex items-center justify-center text-white mb-3 sm:mb-4 group-hover:scale-110 transition-transform">
                {feature.icon}
              </div>
              <h3 className="text-lg sm:text-xl font-semibold text-cream mb-2">{feature.title}</h3>
              <p className="text-cream/60 text-sm sm:text-base leading-relaxed">{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
