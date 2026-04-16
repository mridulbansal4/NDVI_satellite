import React, { useState, useRef, useEffect, useCallback } from 'react';
import KrishiMitraPanel from './KrishiMitraPanel';

export default function Sidebar({ analysisData, activeField, collapsed = false }) {
    const [width, setWidth] = useState(400); 
    const isResizing = useRef(false);

    const handleMouseMove = useCallback((e) => {
        if (!isResizing.current) return;
        let newWidth = e.clientX;
        if (newWidth < 300) newWidth = 300; 
        if (newWidth > 800) newWidth = 800; 
        setWidth(newWidth);
    }, []);

    const handleMouseUp = useCallback(() => {
        if (isResizing.current) {
            isResizing.current = false;
            document.body.style.cursor = 'default';
        }
    }, []);

    useEffect(() => {
        document.addEventListener('mousemove', handleMouseMove);
        document.addEventListener('mouseup', handleMouseUp);
        return () => {
            document.removeEventListener('mousemove', handleMouseMove);
            document.removeEventListener('mouseup', handleMouseUp);
        };
    }, [handleMouseMove, handleMouseUp]);

    return (
        <aside
            id="farm-assistant-sidebar"
            className={`sidebar${collapsed ? ' sidebar--collapsed' : ''}`}
            role="complementary"
            aria-label="Farm Assistant Panel"
            aria-hidden={collapsed}
            style={{
                width: collapsed ? 0 : `${width}px`,
                minWidth: collapsed ? 0 : undefined,
                flexShrink: 0,
                position: 'relative',
                overflow: 'hidden',
                transition: 'width 0.22s ease, min-width 0.22s ease, opacity 0.18s ease',
                opacity: collapsed ? 0 : 1,
                pointerEvents: collapsed ? 'none' : 'auto',
            }}
        >
            <KrishiMitraPanel analysisData={analysisData} activeField={activeField} />
            {!collapsed && (
                <div
                    onMouseDown={(e) => {
                        e.preventDefault();
                        isResizing.current = true;
                        document.body.style.cursor = 'col-resize';
                    }}
                    title="Drag to resize panel"
                    className="sidebar-resize-handle"
                />
            )}
        </aside>
    );
}
