/**
 * InputField.jsx
 * Props: label, name, type, value, onChange, error, placeholder, required, disabled, hint
 * Reusable input with label + error message. Min height 48px for touch targets.
 */

export default function InputField({
  label,
  name,
  type = 'text',
  value,
  onChange,
  error,
  placeholder = '',
  required = false,
  disabled = false,
  hint = '',
}) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={name} className="text-sm font-medium text-[#e2e8f0]">
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </label>

      <input
        id={name}
        name={name}
        type={type}
        value={value}
        onChange={onChange}
        placeholder={placeholder}
        required={required}
        disabled={disabled}
        className={`w-full min-h-[48px] px-4 py-3 text-base rounded-lg border transition-colors duration-150
          ${error
            ? 'border-red-400 focus:ring-2 focus:ring-red-300 focus:border-red-400'
            : 'border-[#ffffff1a] focus:ring-2 focus:ring-green-400 focus:border-green-400'
          }
          ${disabled ? 'bg-[#1e2d3d] text-[#7a90a8] cursor-not-allowed' : 'bg-[#111820] text-white'}
          outline-none`}
      />

      {hint && !error && (
        <p className="text-xs text-[#4a5568]">{hint}</p>
      )}
      {error && (
        <p className="text-xs text-red-500 flex items-center gap-1">
          <svg className="w-3.5 h-3.5" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7 4a1 1 0 11-2 0 1 1 0 012 0zm-1-9a1 1 0 00-1 1v4a1 1 0 102 0V6a1 1 0 00-1-1z" clipRule="evenodd" />
          </svg>
          {error}
        </p>
      )}
    </div>
  );
}
