/**
 * PrimaryButton.jsx
 * Props: label, onClick, disabled, loading, type, variant ("primary" | "outline")
 * Large, full-width, touch-friendly button.
 */

export default function PrimaryButton({
  label,
  onClick,
  disabled = false,
  loading = false,
  type = 'button',
  variant = 'primary',
}) {
  const base = 'h-12 w-full rounded-xl font-semibold text-base transition-all duration-200 flex items-center justify-center gap-2 focus:outline-none focus:ring-2 focus:ring-offset-2';
  const primary = 'bg-[#1A6B3C] hover:bg-[#155c33] active:bg-green-800 text-white focus:ring-green-500 disabled:bg-[#1a2432] disabled:text-[#7a90a8] disabled:cursor-not-allowed';
  const outline = 'border-2 border-[#1A6B3C] text-[#1A6B3C] hover:bg-[#1A6B3C]/10 active:bg-green-100 focus:ring-green-500 disabled:border-[#ffffff1a] disabled:text-[#4a5568] disabled:cursor-not-allowed';

  return (
    <button
      type={type}
      onClick={onClick}
      disabled={disabled || loading}
      className={`${base} ${variant === 'outline' ? outline : primary}`}
    >
      {loading && (
        <svg className="animate-spin h-5 w-5" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4" />
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
      )}
      {loading ? 'Please wait...' : label}
    </button>
  );
}
