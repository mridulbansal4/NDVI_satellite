import React, { useState, useRef, useEffect } from 'react';

export default function NavbarDropdown({ value, options, onChange, style }) {
    const [isOpen, setIsOpen] = useState(false);
    const dropdownRef = useRef(null);

    useEffect(() => {
        const handleClickOutside = (event) => {
            if (dropdownRef.current && !dropdownRef.current.contains(event.target)) {
                setIsOpen(false);
            }
        };
        document.addEventListener('mousedown', handleClickOutside);
        return () => document.removeEventListener('mousedown', handleClickOutside);
    }, []);

    const selectedOption = options.find(opt => opt.value === value) || options[0];

    return (
        <div className="navbar__custom-dropdown" ref={dropdownRef} style={style}>
            <div 
                className="navbar__custom-trigger" 
                onClick={() => setIsOpen(!isOpen)}
            >
                <div className="navbar__custom-selected">{selectedOption ? selectedOption.label : 'Select...'}</div>
                <div className="navbar__select-arrow">▼</div>
            </div>
            
            {isOpen && (
                <div className="navbar__custom-menu">
                    {options.map((opt, idx) => (
                        <div 
                            key={opt.value + '-' + idx}
                            className={`navbar__custom-option ${opt.value === value ? 'is-selected' : ''} ${opt.disabled ? 'is-disabled' : ''} ${opt.action ? 'is-action' : ''}`}
                            onClick={() => {
                                if (opt.disabled) return;
                                if (typeof opt.action === 'function') {
                                    opt.action();
                                    setIsOpen(false);
                                    return;
                                }
                                if (onChange) onChange(opt.value);
                                setIsOpen(false);
                            }}
                        >
                            {opt.label}
                        </div>
                    ))}
                </div>
            )}
        </div>
    );
}
