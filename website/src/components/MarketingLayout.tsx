'use client';

import React, { useState } from 'react';
import { Github, ExternalLink, Menu, X } from 'lucide-react';
import Image from 'next/image';

interface MarketingNavProps {
  transparent?: boolean;
}

export function MarketingNav({ transparent = false }: MarketingNavProps) {
  const [mobileMenuOpen, setMobileMenuOpen] = useState(false);

  const handleSmoothScroll = (e: React.MouseEvent<HTMLAnchorElement>, targetId: string) => {
    e.preventDefault();
    document.querySelector(targetId)?.scrollIntoView({ behavior: 'smooth' });
    setMobileMenuOpen(false);
  };

  return (
    <nav className={`fixed left-0 right-0 top-0 z-50 w-full px-4 sm:px-6 py-4 backdrop-blur-xl transition-colors ${transparent ? 'bg-dark-slate/80' : 'bg-dark-slate border-b border-prr-amber/30'}`}>
      <div className="max-w-7xl mx-auto flex items-center justify-between">
        <a href="/" className="hover:opacity-80 transition-opacity flex items-center gap-2 sm:gap-3">
          <Image src="/favicon.svg" alt="prr" width={32} height={32} className="w-7 h-7 sm:w-8 sm:h-8" />
          <span className="text-lg sm:text-xl font-bold text-cream">prr</span>
        </a>

        {/* Desktop Navigation */}
        <div className="hidden lg:flex items-center gap-8">
          <a
            href="/#features"
            onClick={(e) => handleSmoothScroll(e, '#features')}
            className="text-cream/80 hover:text-cream transition-colors text-sm font-medium"
          >
            Features
          </a>
          <a
            href="/#how-it-works"
            onClick={(e) => handleSmoothScroll(e, '#how-it-works')}
            className="text-cream/80 hover:text-cream transition-colors text-sm font-medium"
          >
            How It Works
          </a>
          <a
            href="/#use-cases"
            onClick={(e) => handleSmoothScroll(e, '#use-cases')}
            className="text-cream/80 hover:text-cream transition-colors text-sm font-medium"
          >
            Use Cases
          </a>
          <div className="flex items-center gap-3 ml-2">
            <a
              href="https://github.com/dotbrains/prr"
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 px-4 py-2 bg-dark-gray hover:bg-dark-slate border border-prr-amber text-cream rounded-lg transition-colors text-sm font-medium"
            >
              <Github className="w-4 h-4" />
              <span>Star</span>
            </a>
            <a
              href="/#quick-start"
              onClick={(e) => handleSmoothScroll(e, '#quick-start')}
              className="bg-gradient-to-r from-prr-amber to-prr-orange hover:from-prr-orange hover:to-prr-gold text-white px-6 py-2 rounded-lg shadow-lg shadow-prr-amber/30 text-sm font-semibold transition-all"
            >
              Get Started
            </a>
          </div>
        </div>

        {/* Mobile Menu Button */}
        <button
          onClick={() => setMobileMenuOpen(!mobileMenuOpen)}
          className="lg:hidden p-2 text-cream hover:text-prr-amber transition-colors"
          aria-label="Toggle menu"
        >
          {mobileMenuOpen ? <X className="w-6 h-6" /> : <Menu className="w-6 h-6" />}
        </button>
      </div>

      {/* Mobile Menu */}
      {mobileMenuOpen && (
        <div className="lg:hidden fixed top-[72px] left-0 right-0 bg-dark-slate border-b border-prr-amber/30 backdrop-blur-xl shadow-xl z-40">
          <div className="px-4 py-6 space-y-4">
            <a
              href="/#features"
              onClick={(e) => handleSmoothScroll(e, '#features')}
              className="block text-cream/80 hover:text-cream transition-colors text-base font-medium py-2"
            >
              Features
            </a>
            <a
              href="/#how-it-works"
              onClick={(e) => handleSmoothScroll(e, '#how-it-works')}
              className="block text-cream/80 hover:text-cream transition-colors text-base font-medium py-2"
            >
              How It Works
            </a>
            <a
              href="/#use-cases"
              onClick={(e) => handleSmoothScroll(e, '#use-cases')}
              className="block text-cream/80 hover:text-cream transition-colors text-base font-medium py-2"
            >
              Use Cases
            </a>
            <div className="pt-4 space-y-3 border-t border-prr-amber/20">
              <a
                href="https://github.com/dotbrains/prr"
                target="_blank"
                rel="noopener noreferrer"
                className="flex items-center justify-center gap-2 px-4 py-3 bg-dark-gray hover:bg-dark-slate border border-prr-amber text-cream rounded-lg transition-colors text-sm font-medium w-full"
              >
                <Github className="w-4 h-4" />
                <span>Star on GitHub</span>
              </a>
              <a
                href="/#quick-start"
                onClick={(e) => handleSmoothScroll(e, '#quick-start')}
                className="flex items-center justify-center bg-gradient-to-r from-prr-amber to-prr-orange hover:from-prr-orange hover:to-prr-gold text-white px-6 py-3 rounded-lg shadow-lg shadow-prr-amber/30 text-sm font-semibold transition-all w-full"
              >
                Get Started
              </a>
            </div>
          </div>
        </div>
      )}
    </nav>
  );
}

export function MarketingFooter() {
  const handleSmoothScroll = (e: React.MouseEvent<HTMLAnchorElement>, targetId: string) => {
    e.preventDefault();
    document.querySelector(targetId)?.scrollIntoView({ behavior: 'smooth' });
  };

  return (
    <footer className="bg-dark-slate border-t border-prr-amber/30 py-12 sm:py-16">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-8 sm:gap-12 mb-12">
          <div>
            <div className="flex items-center gap-3 mb-4">
              <Image src="/favicon.svg" alt="prr" width={32} height={32} className="w-8 h-8" />
              <span className="text-xl font-bold text-cream">prr</span>
            </div>
            <p className="text-cream/60 text-sm leading-relaxed mb-4">
              AI-powered PR code review CLI. Human-like comments, structured output, one command. Open source and free to use.
            </p>
            <div className="flex items-center gap-3">
              <a
                href="https://github.com/dotbrains/prr"
                className="text-cream/60 hover:text-cream transition-colors"
                target="_blank"
                rel="noopener noreferrer"
                aria-label="GitHub"
              >
                <Github className="w-5 h-5" />
              </a>
            </div>
          </div>
          <div>
            <h4 className="text-cream font-semibold mb-4 text-sm uppercase tracking-wider">Product</h4>
            <ul className="space-y-3">
              <li>
                <a
                  href="/#features"
                  onClick={(e) => handleSmoothScroll(e, '#features')}
                  className="text-cream/70 hover:text-cream text-sm transition-colors inline-block cursor-pointer"
                >
                  Features
                </a>
              </li>
              <li>
                <a
                  href="/#how-it-works"
                  onClick={(e) => handleSmoothScroll(e, '#how-it-works')}
                  className="text-cream/70 hover:text-cream text-sm transition-colors inline-block cursor-pointer"
                >
                  How It Works
                </a>
              </li>
              <li>
                <a
                  href="/#use-cases"
                  onClick={(e) => handleSmoothScroll(e, '#use-cases')}
                  className="text-cream/70 hover:text-cream text-sm transition-colors inline-block cursor-pointer"
                >
                  Use Cases
                </a>
              </li>
              <li>
                <a
                  href="/#quick-start"
                  onClick={(e) => handleSmoothScroll(e, '#quick-start')}
                  className="text-cream/70 hover:text-cream text-sm transition-colors inline-block cursor-pointer"
                >
                  Quick Start
                </a>
              </li>
            </ul>
          </div>
          <div>
            <h4 className="text-cream font-semibold mb-4 text-sm uppercase tracking-wider">Resources</h4>
            <ul className="space-y-3">
              <li>
                <a href="https://github.com/dotbrains/prr#readme" className="text-cream/70 hover:text-cream text-sm transition-colors inline-flex items-center gap-1.5" target="_blank" rel="noopener noreferrer">
                  Documentation
                  <ExternalLink className="w-3 h-3" />
                </a>
              </li>
              <li>
                <a href="https://github.com/dotbrains/prr/blob/master/SPEC.md" className="text-cream/70 hover:text-cream text-sm transition-colors inline-flex items-center gap-1.5" target="_blank" rel="noopener noreferrer">
                  Specification
                  <ExternalLink className="w-3 h-3" />
                </a>
              </li>
            </ul>
          </div>
          <div>
            <h4 className="text-cream font-semibold mb-4 text-sm uppercase tracking-wider">Community</h4>
            <ul className="space-y-3">
              <li>
                <a href="https://github.com/dotbrains/prr" className="text-cream/70 hover:text-cream text-sm transition-colors inline-flex items-center gap-1.5" target="_blank" rel="noopener noreferrer">
                  GitHub Repository
                  <ExternalLink className="w-3 h-3" />
                </a>
              </li>
              <li>
                <a href="https://github.com/dotbrains/prr/issues" className="text-cream/70 hover:text-cream text-sm transition-colors inline-flex items-center gap-1.5" target="_blank" rel="noopener noreferrer">
                  Report Issues
                  <ExternalLink className="w-3 h-3" />
                </a>
              </li>
              <li>
                <a href="https://github.com/dotbrains/prr/discussions" className="text-cream/70 hover:text-cream text-sm transition-colors inline-flex items-center gap-1.5" target="_blank" rel="noopener noreferrer">
                  Discussions
                  <ExternalLink className="w-3 h-3" />
                </a>
              </li>
            </ul>
          </div>
        </div>
        <div className="pt-8 border-t border-prr-amber/30 flex flex-col md:flex-row justify-between items-center gap-4">
          <p className="text-cream/60 text-sm">
            © {new Date().getFullYear()} prr. All rights reserved.
          </p>
          <p className="text-cream/50 text-xs">
            Open source software licensed under MIT
          </p>
        </div>
      </div>
    </footer>
  );
}
