import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../../context/AuthContext';
import { useToast } from '../../context/ToastContext';
import { AI_API_BASE_URL } from '../../services/api';

interface AdminAIChatDrawerProps {
  isOpen: boolean;
  onClose: () => void;
}

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  toolCalls?: Array<{ name: string; args: any }>;
}

const AdminFormattedMessage: React.FC<{ text: string }> = ({ text }) => {
  const lines = text.split('\n');

  return (
    <div className="space-y-1.5 leading-relaxed text-xs sm:text-sm">
      {lines.map((line, lineIdx) => {
        const trimmed = line.trim();
        if (!trimmed) return <div key={lineIdx} className="h-1.5" />;

        // Header check
        if (trimmed.startsWith('### ')) {
          return (
            <h4 key={lineIdx} className="font-bold text-xs sm:text-sm text-purple-900 mt-2">
              {trimmed.replace(/^###\s+/, '')}
            </h4>
          );
        }

        // Bullet lists
        const isBullet = trimmed.startsWith('- ') || trimmed.startsWith('* ') || trimmed.startsWith('• ');
        const rawContent = isBullet ? trimmed.replace(/^[-*•]\s+/, '') : trimmed;

        // Numbered lists
        const numMatch = rawContent.match(/^(\d+)\.\s+(.*)/);
        const listPrefix = numMatch ? `${numMatch[1]}. ` : isBullet ? '• ' : '';
        const textContent = numMatch ? numMatch[2] : rawContent;

        // Split by bold (**text**) and code (`code`)
        const parts = textContent.split(/(\*\*.*?\*\*|`.*?`)/g);

        return (
          <div key={lineIdx} className={`${isBullet || numMatch ? 'flex items-start gap-1.5 ml-1' : ''}`}>
            {(isBullet || numMatch) && (
              <span className="font-bold text-purple-600 shrink-0">{listPrefix}</span>
            )}
            <div className="flex-1">
              {parts.map((part, partIdx) => {
                if (part.startsWith('**') && part.endsWith('**')) {
                  return (
                    <strong key={partIdx} className="font-bold text-slate-900">
                      {part.slice(2, -2)}
                    </strong>
                  );
                }
                if (part.startsWith('`') && part.endsWith('`')) {
                  return (
                    <code
                      key={partIdx}
                      className="px-1.5 py-0.5 rounded bg-purple-50 text-purple-700 font-mono text-[11px] border border-purple-200 font-semibold"
                    >
                      {part.slice(1, -1)}
                    </code>
                  );
                }
                return <span key={partIdx}>{part}</span>;
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export const AdminAIChatDrawer: React.FC<AdminAIChatDrawerProps> = ({ isOpen, onClose }) => {
  const { t, i18n } = useTranslation();
  const { token, currentUser } = useAuth();
  const { showToast } = useToast();

  const getWelcomeMessage = (lang: string) => {
    return lang === 'en'
      ? `Hello Admin **${currentUser?.name || 'Manager'}** 👋\n\nI am your **Tirenn Admin AI Copilot**.\n\nI can help you analyze **Revenue KPIs**, identify **Low Stock Items**, execute **Stock Adjustments**, and consult confidential **Warehouse SOP & Audit Protocols**.`
      : `Halo Admin **${currentUser?.name || 'Manager'}** 👋\n\nSaya adalah **Tirenn Admin AI Copilot**.\n\nSaya dapat membantu Anda memantau **Metrik Omzet Toko**, mendeteksi **Stok Menipis**, mengeksekusi **Penyesuaian Stok**, dan membuka dokumen rahasia **SOP Gudang & Audit Inventaris**.`;
  };

  const [messages, setMessages] = useState<Message[]>([
    {
      id: 'welcome',
      role: 'assistant',
      content: getWelcomeMessage(i18n.language),
    },
  ]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const [sessionId, setSessionId] = useState<string>(() => {
    const saved = localStorage.getItem('tirenn_admin_ai_session_id');
    if (saved) return saved;
    const newId = `admin_session_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
    localStorage.setItem('tirenn_admin_ai_session_id', newId);
    return newId;
  });

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    if (isOpen) {
      scrollToBottom();
    }
  }, [messages, isOpen]);

  // Update initial welcome message upon language toggle
  useEffect(() => {
    setMessages((prev) =>
      prev.map((m) =>
        m.id === 'welcome'
          ? {
              ...m,
              content: getWelcomeMessage(i18n.language),
            }
          : m
      )
    );
  }, [i18n.language, currentUser?.name]);

  const handleResetChat = async () => {
    if (sessionId) {
      try {
        await fetch(`${AI_API_BASE_URL}/chat/session/${sessionId}`, {
          method: 'DELETE',
        });
      } catch (err) {
        console.warn('Failed to delete Redis admin chat session:', err);
      }
    }
    const newSessionId = `admin_session_${Date.now()}_${Math.random().toString(36).substring(2, 9)}`;
    localStorage.setItem('tirenn_admin_ai_session_id', newSessionId);
    setSessionId(newSessionId);

    setMessages([
      {
        id: 'welcome',
        role: 'assistant',
        content: getWelcomeMessage(i18n.language),
      },
    ]);
    setInput('');
    showToast(t('admin_copilot.history_cleared'), 'info');
  };

  const handleSend = async (textToSend?: string) => {
    const query = textToSend || input.trim();
    if (!query || loading) return;

    if (!token) {
      showToast(t('admin_copilot.err_auth_required'), 'error');
      return;
    }

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: query,
    };

    setMessages((prev) => [...prev, userMsg]);
    if (!textToSend) setInput('');
    setLoading(true);

    try {
      const res = await fetch(`${AI_API_BASE_URL}/chat/admin`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`,
        },
        body: JSON.stringify({
          messages: [...messages, userMsg].map((m) => ({
            role: m.role,
            content: m.content,
          })),
          session_id: sessionId,
        }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.detail || t('admin_copilot.err_general'));
      }

      const data = await res.json();
      const aiMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: data.reply || '',
        toolCalls: data.tool_calls || [],
      };

      setMessages((prev) => [...prev, aiMsg]);
    } catch (err: any) {
      showToast(err.message || t('admin_copilot.err_general'), 'error');
      setMessages((prev) => [
        ...prev,
        {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: `⚠️ ${err.message || t('admin_copilot.err_general')}`,
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-50 overflow-hidden">
      {/* Backdrop */}
      <div
        className="fixed inset-0 bg-slate-900/40 backdrop-blur-xs transition-opacity animate-fade-in"
        onClick={onClose}
      />

      {/* Slide-over Drawer Panel */}
      <div className="fixed inset-y-0 right-0 max-w-full flex pl-10">
        <div className="w-screen max-w-md sm:max-w-lg bg-white shadow-2xl flex flex-col border-l border-slate-200">
          
          {/* Header */}
          <div className="p-4 sm:p-5 bg-gradient-to-r from-slate-900 via-purple-950 to-slate-900 text-white flex items-center justify-between border-b border-purple-800/40 shrink-0">
            <div className="flex items-center gap-3">
              <div className="w-10 h-10 rounded-xl bg-purple-600/30 border border-purple-500/40 flex items-center justify-center text-xl shadow-inner">
                ⚡
              </div>
              <div>
                <h3 className="font-bold text-sm sm:text-base flex items-center gap-2">
                  <span>{t('admin_copilot.title')}</span>
                  <span className="text-[10px] font-mono px-2 py-0.5 rounded-full bg-purple-500/20 text-purple-300 border border-purple-500/30 font-semibold">
                    Admin Only
                  </span>
                </h3>
                <p className="text-[11px] text-purple-200/80 line-clamp-1">
                  {t('admin_copilot.subtitle')}
                </p>
              </div>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={handleResetChat}
                className="text-slate-300 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors text-xs font-semibold flex items-center gap-1 cursor-pointer"
                title="Reset conversation"
              >
                <span>🔄</span>
              </button>
              <button
                onClick={onClose}
                className="text-slate-300 hover:text-white p-2 rounded-lg hover:bg-white/10 transition-colors text-base cursor-pointer"
              >
                ✕
              </button>
            </div>
          </div>

          {/* Messages Scroll Area */}
          <div className="flex-1 overflow-y-auto p-4 sm:p-5 space-y-4 bg-slate-50/50">
            {messages.map((m) => {
              const isUser = m.role === 'user';
              return (
                <div
                  key={m.id}
                  className={`flex flex-col ${isUser ? 'items-end' : 'items-start'} space-y-1.5`}
                >
                  <div className="flex items-center gap-1.5 px-1 text-[11px] font-bold text-slate-400">
                    <span>{isUser ? '👤 Admin' : '⚡ Admin AI Copilot'}</span>
                  </div>

                  <div
                    className={`p-3.5 rounded-2xl text-xs sm:text-sm leading-relaxed max-w-[88%] shadow-xs ${
                      isUser
                        ? 'bg-purple-700 text-white rounded-br-xs font-medium'
                        : 'bg-white border border-slate-200 text-slate-800 rounded-bl-xs'
                    }`}
                  >
                    {/* Tool Badges if executed */}
                    {m.toolCalls && m.toolCalls.length > 0 && (
                      <div className="mb-2 pb-2 border-b border-slate-100 flex flex-wrap gap-1.5">
                        {m.toolCalls.map((tc, idx) => (
                          <span
                            key={idx}
                            className="inline-flex items-center gap-1 px-2 py-0.5 rounded-md font-mono text-[10px] font-semibold bg-emerald-50 text-emerald-700 border border-emerald-200"
                          >
                            <span>⚙️</span>
                            <span>{tc.name}</span>
                          </span>
                        ))}
                      </div>
                    )}

                    {isUser ? (
                      <div className="whitespace-pre-line break-words">{m.content}</div>
                    ) : (
                      <AdminFormattedMessage text={m.content} />
                    )}
                  </div>
                </div>
              );
            })}

            {loading && (
              <div className="flex items-start gap-2">
                <div className="bg-white border border-slate-200 rounded-2xl rounded-bl-xs p-3.5 shadow-xs text-xs text-slate-500 flex items-center gap-2">
                  <span className="animate-spin text-purple-600">⚡</span>
                  <span className="font-medium animate-pulse">{t('admin_copilot.thinking')}</span>
                </div>
              </div>
            )}

            <div ref={messagesEndRef} />
          </div>

          {/* Quick Action Suggestion Chips */}
          <div className="p-3 bg-white border-t border-slate-100 flex items-center gap-1.5 overflow-x-auto scrollbar-none shrink-0">
            <button
              onClick={() => handleSend(t('admin_copilot.quick_metrics'))}
              className="px-2.5 py-1.5 rounded-lg bg-slate-100 hover:bg-purple-50 hover:text-purple-700 text-slate-700 font-semibold text-[11px] transition-all whitespace-nowrap shrink-0 border border-slate-200/80 cursor-pointer"
            >
              {t('admin_copilot.quick_metrics')}
            </button>
            <button
              onClick={() => handleSend(t('admin_copilot.quick_low_stock'))}
              className="px-2.5 py-1.5 rounded-lg bg-slate-100 hover:bg-purple-50 hover:text-purple-700 text-slate-700 font-semibold text-[11px] transition-all whitespace-nowrap shrink-0 border border-slate-200/80 cursor-pointer"
            >
              {t('admin_copilot.quick_low_stock')}
            </button>
            <button
              onClick={() => handleSend(t('admin_copilot.quick_sop_picking'))}
              className="px-2.5 py-1.5 rounded-lg bg-slate-100 hover:bg-purple-50 hover:text-purple-700 text-slate-700 font-semibold text-[11px] transition-all whitespace-nowrap shrink-0 border border-slate-200/80 cursor-pointer"
            >
              {t('admin_copilot.quick_sop_picking')}
            </button>
            <button
              onClick={() => handleSend(t('admin_copilot.quick_recent_orders'))}
              className="px-2.5 py-1.5 rounded-lg bg-slate-100 hover:bg-purple-50 hover:text-purple-700 text-slate-700 font-semibold text-[11px] transition-all whitespace-nowrap shrink-0 border border-slate-200/80 cursor-pointer"
            >
              {t('admin_copilot.quick_recent_orders')}
            </button>
          </div>

          {/* Input Form */}
          <div className="p-4 bg-white border-t border-slate-200 shrink-0">
            <form
              onSubmit={(e) => {
                e.preventDefault();
                handleSend();
              }}
              className="flex items-center gap-2"
            >
              <input
                type="text"
                value={input}
                onChange={(e) => setInput(e.target.value)}
                placeholder={t('admin_copilot.placeholder')}
                disabled={loading}
                className="flex-1 text-xs sm:text-sm bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 outline-none focus:bg-white focus:border-purple-600 font-medium transition-all"
              />
              <button
                type="submit"
                disabled={loading || !input.trim()}
                className="px-4 py-2.5 bg-purple-700 hover:bg-purple-800 disabled:bg-slate-200 disabled:text-slate-400 text-white font-bold text-xs sm:text-sm rounded-xl transition-all shadow-xs flex items-center gap-1.5 cursor-pointer"
              >
                <span>🚀</span>
                <span className="hidden sm:inline">{t('admin_copilot.send')}</span>
              </button>
            </form>
          </div>

        </div>
      </div>
    </div>
  );
};
