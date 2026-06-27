/**
 * ProgressBar.jsx
 * Props: currentStep (number), totalSteps (number, default 9)
 * Fixed at top of screen, green fill, shows "Step X of 9"
 */

export default function ProgressBar({ currentStep, totalSteps = 9 }) {
  const pct = Math.min(100, Math.round((currentStep / totalSteps) * 100));
  return (
    <div className="fixed top-0 left-0 right-0 z-50">
      {/* Progress bar track */}
      <div className="h-1.5 bg-gray-200 w-full">
        <div
          className="h-1.5 bg-[#1A6B3C]/100 transition-all duration-500 ease-out"
          style={{ width: `${pct}%` }}
        />
      </div>
      {/* Step label */}
      <div className="bg-[#111820]/90 backdrop-blur-sm border-b border-[#ffffff1a] px-4 py-2 flex items-center justify-between">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded-full bg-[#1A6B3C]/100 flex items-center justify-center">
            <span className="text-white text-xs font-bold">{currentStep}</span>
          </div>
          <span className="text-sm font-medium text-[#e2e8f0]">
            Step {currentStep} of {totalSteps}
          </span>
        </div>
        <div className="flex gap-1">
          {Array.from({ length: totalSteps }).map((_, i) => (
            <div
              key={i}
              className={`w-2 h-2 rounded-full transition-all duration-300 ${
                i < currentStep ? 'bg-[#1A6B3C]/100' : 'bg-gray-200'
              }`}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
