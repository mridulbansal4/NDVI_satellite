import React, { useEffect, useState } from 'react';

/**
 * Modal to collect or edit a field name (after draw, or rename from menu).
 */
export default function FieldNameModal({
  open,
  title,
  description,
  initialName,
  confirmLabel = 'Continue',
  cancelLabel = 'Cancel',
  onConfirm,
  onCancel,
}) {
  const [value, setValue] = useState(initialName || '');

  useEffect(() => {
    if (open) setValue(initialName || '');
  }, [open, initialName]);

  if (!open) return null;

  const submit = () => {
    const trimmed = value.trim();
    if (!trimmed) return;
    onConfirm(trimmed);
  };

  return (
    <div className="field-name-modal-backdrop" role="presentation" onMouseDown={onCancel}>
      <div
        className="field-name-modal"
        role="dialog"
        aria-modal="true"
        aria-labelledby="field-name-modal-title"
        onMouseDown={(e) => e.stopPropagation()}
      >
        <h2 id="field-name-modal-title" className="field-name-modal__title">
          {title}
        </h2>
        {description && <p className="field-name-modal__desc">{description}</p>}
        <label className="field-name-modal__label" htmlFor="field-name-input">
          Field name
        </label>
        <input
          id="field-name-input"
          className="field-name-modal__input"
          type="text"
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === 'Enter') submit();
            if (e.key === 'Escape') onCancel();
          }}
          autoFocus
          maxLength={80}
          autoComplete="off"
        />
        <div className="field-name-modal__actions">
          <button type="button" className="btn btn--ghost btn--sm" onClick={onCancel}>
            {cancelLabel}
          </button>
          <button
            type="button"
            className="btn btn--primary btn--sm"
            onClick={submit}
            disabled={!value.trim()}
          >
            {confirmLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
