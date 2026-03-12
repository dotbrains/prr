'use client';

export function StatsSection() {
  return (
    <section className="py-12 sm:py-16 bg-dark-gray/50">
      <div className="max-w-7xl mx-auto px-4 sm:px-6">
        <div className="text-center mb-8 sm:mb-12">
          <h2 className="text-2xl sm:text-3xl lg:text-4xl font-bold text-cream mb-3 sm:mb-4">
            Code reviews that sound like your team wrote them
          </h2>
          <p className="text-cream/70 text-base sm:text-lg lg:text-xl">
            Local-first, provider-agnostic, and designed for real engineering workflows
          </p>
        </div>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-4 sm:gap-6 md:gap-8 text-center">
          <div>
            <div className="text-2xl sm:text-3xl font-bold text-gradient mb-1 sm:mb-2">PR + Local</div>
            <div className="text-cream/60 text-sm sm:text-base">Review Modes</div>
          </div>
          <div>
            <div className="text-2xl sm:text-3xl font-bold text-gradient mb-1 sm:mb-2">Claude</div>
            <div className="text-cream/60 text-sm sm:text-base">Opus Default</div>
          </div>
          <div>
            <div className="text-2xl sm:text-3xl font-bold text-gradient mb-1 sm:mb-2">Markdown</div>
            <div className="text-cream/60 text-sm sm:text-base">Structured Output</div>
          </div>
          <div>
            <div className="text-2xl sm:text-3xl font-bold text-gradient mb-1 sm:mb-2">Zero</div>
            <div className="text-cream/60 text-sm sm:text-base">Bot Comments</div>
          </div>
        </div>
      </div>
    </section>
  );
}
