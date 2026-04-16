import React, { useState, useRef, useEffect, useCallback } from 'react';
import { Sprout, MessageSquareText, Leaf } from 'lucide-react';
import { apiUrl } from './api';

const CHATBOT_API = apiUrl('/chatbot/chat');

// ─────────────────────────────────────────────────────────────────────────────
// TIMESTAMP HELPER
// ─────────────────────────────────────────────────────────────────────────────
function formatTime(date) {
  return date.toLocaleTimeString('en-IN', {
    hour:   '2-digit',
    minute: '2-digit',
    hour12: true,
  });
}

// ─────────────────────────────────────────────────────────────────────────────
// TYPING INDICATOR
// ─────────────────────────────────────────────────────────────────────────────
function TypingIndicator() {
  return (
    <div className="km-message km-message--assistant">
      <div className="km-avatar" aria-hidden="true">
        <MessageSquareText size={12} strokeWidth={2.25} />
      </div>
      <div className="km-bubble km-bubble--assistant">
        <div className="km-typing-dots">
          <span className="km-dot km-dot--1" />
          <span className="km-dot km-dot--2" />
          <span className="km-dot km-dot--3" />
        </div>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// SKELETON LOADER
// ─────────────────────────────────────────────────────────────────────────────
function SkeletonLoader() {
  return (
    <div className="km-skeleton-wrap">
      <div className="km-skeleton-row km-skeleton-row--wide" />
      <div className="km-skeleton-row km-skeleton-row--med" />
      <div className="km-skeleton-row km-skeleton-row--narrow" />
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// MESSAGE BUBBLE
// ─────────────────────────────────────────────────────────────────────────────
function MessageBubble({ msg }) {
  const isAssistant = msg.role === 'assistant';

  const formatContent = (text) => {
    if (!text) return { __html: '' };
    let html = text
      .replace(/\*\*(.*?)\*\*/g, '<strong class="km-strong">$1</strong>')
      // Map asterisk bullet points cleanly
      .replace(/\n\*\s(.*?)/g, '<br/>• $1')
      // Single newlines to line breaks
      .replace(/\n/g, '<br/>');
    
    return { __html: html };
  };

  return (
    <div className={`km-message km-message--${msg.role}`}>
      {isAssistant && (
        <div className="km-avatar" aria-hidden="true">
          <MessageSquareText size={12} strokeWidth={2.25} />
        </div>
      )}
      <div className={`km-bubble km-bubble--${msg.role}`}>
        <div 
            className="km-bubble__text" 
            dangerouslySetInnerHTML={formatContent(msg.content)} 
            style={{ whiteSpace: 'normal', lineHeight: '1.6' }} 
        />
        <span className="km-bubble__time">{formatTime(msg.timestamp)}</span>
      </div>
    </div>
  );
}

// ─────────────────────────────────────────────────────────────────────────────
// MAIN PANEL
// ─────────────────────────────────────────────────────────────────────────────
export default function KrishiMitraPanel({ analysisData, activeField }) {
  const [messages,       setMessages]       = useState([]);
  const [inputValue,     setInputValue]     = useState('');
  const [isLoading,      setIsLoading]      = useState(false);
  const [isInitializing, setIsInitializing] = useState(true);
  const [ollamaStatus,   setOllamaStatus]   = useState('connecting');

  // Stable session ID for the lifetime of this panel mount
  const sessionIdRef    = useRef(crypto.randomUUID());
  const messagesEndRef  = useRef(null);

  // Auto-scroll after every state change that adds messages
  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  }, [messages, isLoading]);

  // ── Core fetch helper ───────────────────────────────────────────────────────
  const callChatAPI = useCallback(async (message, context = null) => {
    const reqBody = {
      session_id: sessionIdRef.current,
      message,
    };
    if (context) {
      reqBody.farmData = context.farmContext;
      reqBody.heatmapData = context.heatmapContext;
    }

    const res = await fetch(CHATBOT_API, {
      method:  'POST',
      headers: { 'Content-Type': 'application/json' },
      body:    JSON.stringify(reqBody),
    });

    if (!res.ok) {
      const err = await res.json().catch(() => ({}));
      throw new Error(err.error || `HTTP ${res.status}`);
    }

    const data = await res.json();
    if (!data.reply) throw new Error('Empty reply from server.');
    return data.reply;
  }, []);

  // ── Stable Context Calculation ──────────────────────────────────────────────
  const activeContext = React.useMemo(() => {
    if (!analysisData || !activeField) return null;

    const cells = analysisData.features || [];
    let stressed = 0, moderate = 0, healthy = 0;
    let sumCvi = 0, sumNdvi = 0, sumEvi = 0, sumSavi = 0, sumNdmi = 0, sumGndvi = 0;

    cells.forEach(f => {
       const p = f.properties || {};
       const v = p.cvi || p.CVI || 0;
       if (v < 0.3) stressed++;
       else if (v < 0.6) moderate++;
       else healthy++;

       sumCvi += v;
       sumNdvi += p.ndvi || p.NDVI || 0;
       sumEvi += p.evi || p.EVI || 0;
       sumSavi += p.savi || p.SAVI || 0;
       sumNdmi += p.ndmi || p.NDMI || 0;
       sumGndvi += p.gndvi || p.GNDVI || 0;
    });

    const total = cells.length || 1;
    const summary = analysisData.farm_summary || {};
    
    return {
      farmContext: {
         fieldName: activeField.name || "Selected Field",
         area: summary.area_ha ? summary.area_ha.toFixed(2) : 0,
         date: analysisData.date || "Today",
         confidence: summary.confidence ? summary.confidence.toFixed(1) : 0,
         cleanScenes: summary.scene_count || 0,
         cvi: summary.cvi_avg ?? (sumCvi / total),
         ndvi: summary.ndvi_avg ?? (sumNdvi / total),
         evi: summary.evi_avg ?? (sumEvi / total),
         savi: summary.savi_avg ?? (sumSavi / total),
         ndmi: summary.ndmi_avg ?? (sumNdmi / total),
         gndvi: summary.gndvi_avg ?? (sumGndvi / total),
      },
      heatmapContext: {
         stressedPct: Math.round((stressed / total) * 100),
         stressedLocation: "the field",
         moderatePct: Math.round((moderate / total) * 100),
         moderateLocation: "the field",
         healthyPct: Math.round((healthy / total) * 100),
         healthyLocation: "the field"
      }
    };
  }, [analysisData, activeField]);


  // ── Auto-summary when farm becomes available ────────────────────────────────
  useEffect(() => {
    if (!activeContext) {
      setMessages([]);
      setIsInitializing(false);
      setOllamaStatus('connecting'); // Or Idle
      return;
    }

    // Reset session for new farm
    sessionIdRef.current = crypto.randomUUID();
    setMessages([]);
    setIsInitializing(true);
    setOllamaStatus('connecting');

    (async () => {
      try {
        const reply = await callChatAPI('Generate the farm summary report now.', activeContext);
        setMessages([{
          id:        Date.now(),
          role:      'assistant',
          content:   reply,
          timestamp: new Date(),
        }]);
        setOllamaStatus('live');
      } catch (err) {
        console.error('AI advisor init error:', err);
        setOllamaStatus('error');
        setMessages([{
          id:        Date.now(),
          role:      'assistant',
          content:   'Could not connect to Ollama (chat assistant). Start Ollama on this PC (default port 11434) and pull the model from chatbot settings. Drawing fields and NDVI analysis only need the Flask backend.',
          timestamp: new Date(),
        }]);
      } finally {
        setIsInitializing(false);
      }
    })();
  }, [activeContext, callChatAPI]);

  // ── Send a user message ─────────────────────────────────────────────────────
  const sendMessage = useCallback(async (text) => {
    const trimmed = text.trim();
    if (!trimmed || isLoading) return;

    const userMsg = {
      id:        Date.now(),
      role:      'user',
      content:   trimmed,
      timestamp: new Date(),
    };

    setMessages(prev => [...prev, userMsg]);
    setInputValue('');
    setIsLoading(true);

    try {
      const reply = await callChatAPI(trimmed, activeContext);
      setMessages(prev => [...prev, {
        id:        Date.now() + 1,
        role:      'assistant',
        content:   reply,
        timestamp: new Date(),
      }]);
    } catch (err) {
      console.error('Chat error:', err);
      setMessages(prev => [...prev, {
        id:        Date.now() + 1,
        role:      'assistant',
        content:   'I could not process that. Please try asking again.',
        timestamp: new Date(),
      }]);
    } finally {
      setIsLoading(false);
    }
  }, [isLoading, callChatAPI, activeContext]);

  const handleKeyDown = (e) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      sendMessage(inputValue);
    }
  };

  // ─────────────────────────────────────────────────────────────────────────
  // RENDER
  // ─────────────────────────────────────────────────────────────────────────
  return (
    <div className="km-panel" aria-label="AI advisor — farm assistant">

      {/* ── Header ── */}
      <header className="km-header">
        <div className="km-header__left">
          <span className="km-header__icon" aria-hidden="true">
            <Sprout size={20} strokeWidth={2} color="var(--c-accent-bright)" />
          </span>
          <div className="km-header__titles">
            <span className="km-header__name">AI advisor</span>
            <span className="km-header__sub">Satellite farm insights</span>
          </div>
        </div>
        <div className={`km-status-badge km-status-badge--${ollamaStatus}`}>
          <span className="km-status-dot" />
          <span className="km-status-label">
            {ollamaStatus === 'connecting' ? 'Connecting'
              : ollamaStatus === 'live'    ? 'Live'
              : 'Offline'}
          </span>
        </div>
      </header>

      {analysisData && activeField && (
        <p className="km-context-strip">
          {activeField.name || 'Field'} · Sentinel-2 indices · advisory from your latest analysis
        </p>
      )}

      {/* ── Messages ── */}
      <section className="km-messages" aria-live="polite">
        {!analysisData || !activeField ? (
          <div className="km-empty">
            <div className="km-empty__icon">
              <Leaf size={20} strokeWidth={1.75} />
            </div>
            Select a field or draw a polygon on the map to load vegetation metrics and advisory.
          </div>
        ) : isInitializing ? (
          <SkeletonLoader />
        ) : (
          messages.map(msg => <MessageBubble key={msg.id} msg={msg} />)
        )}
        {isLoading && !isInitializing && <TypingIndicator />}
        <div ref={messagesEndRef} />
      </section>

      {/* ── Input Row ── */}
      <footer className="km-input-row">
        <input
          id="krishi-mitra-input"
          className="km-input"
          type="text"
          placeholder="Ask about your farm..."
          value={inputValue}
          onChange={e => setInputValue(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={isLoading || isInitializing || !analysisData}
          aria-label="Ask the AI advisor a question"
          autoComplete="off"
        />
        <button
          id="krishi-mitra-send-btn"
          className="km-send-btn"
          onClick={() => sendMessage(inputValue)}
          disabled={isLoading || isInitializing || !analysisData || !inputValue.trim()}
          aria-label="Send message"
        >
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none"
               stroke="currentColor" strokeWidth="2.5"
               strokeLinecap="round" strokeLinejoin="round">
            <line x1="12" y1="19" x2="12" y2="5" />
            <polyline points="5 12 12 5 19 12" />
          </svg>
        </button>
      </footer>
    </div>
  );
}
