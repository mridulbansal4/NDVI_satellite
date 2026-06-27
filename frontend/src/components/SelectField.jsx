/**
 * SelectField.jsx
 * Props: label, name, value, onChange, options [{label, value}], error, required, disabled
 * Reusable select dropdown with label + error message.
 */

export default function SelectField({
  label,
  name,
  value,
  onChange,
  options = [],
  error,
  required = false,
  disabled = false,
  placeholder = 'Select an option',
}) {
  return (
    <div className="flex flex-col gap-1">
      <label htmlFor={name} className="text-sm font-medium text-[#e2e8f0]">
        {label}
        {required && <span className="text-red-500 ml-1">*</span>}
      </label>

      <div className="relative">
        <select
          id={name}
          name={name}
          value={value}
          onChange={onChange}
          required={required}
          disabled={disabled}
          className={`w-full min-h-[48px] px-4 py-3 text-base rounded-lg border appearance-none transition-colors duration-150 pr-10
            ${error
              ? 'border-red-400 focus:ring-2 focus:ring-red-300 focus:border-red-400'
              : 'border-[#ffffff1a] focus:ring-2 focus:ring-green-400 focus:border-green-400'
            }
            ${disabled ? 'bg-[#1e2d3d] text-[#7a90a8] cursor-not-allowed' : 'bg-[#111820] text-white'}
            outline-none`}
        >
          <option value="" disabled>{placeholder}</option>
          {options.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
        {/* Chevron icon */}
        <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-3">
          <svg className="w-5 h-5 text-[#4a5568]" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
          </svg>
        </div>
      </div>

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
