import React, { useState, useRef, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { useCart } from '../context/CartContext';
import { useToast } from '../context/ToastContext';
import { useAuth } from '../context/AuthContext';
import { useCurrency } from '../context/CurrencyContext';
import { AI_API_BASE_URL } from '../services/api';

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  suggestedProducts?: any[];
  toolCalls?: any[];
}

interface AIChatModalProps {
  isOpen: boolean;
  onClose: () => void;
  onAuthRequired?: () => void;
}

// Lightweight Markdown text renderer
const FormattedMessageText: React.FC<{ text: string }> = ({ text }) => {
  const lines = text.split('\n');

  return (
    <div className="space-y-1.5 leading-relaxed">
      {lines.map((line, lineIdx) => {
        const trimmed = line.trim();
        if (!trimmed) return <div key={lineIdx} className="h-1" />;

        // Clean out raw image references
        if (
          trimmed.toLowerCase().startsWith('gambar:') ||
          trimmed.toLowerCase().startsWith('- gambar:') ||
          trimmed.toLowerCase().startsWith('* gambar:') ||
          trimmed.toLowerCase().startsWith('• gambar:') ||
          trimmed.includes('![') ||
          trimmed.includes('[Lihat Gambar]') ||
          trimmed.includes('images.unsplash.com')
        ) {
          return null;
        }

        // Bullet lists
        const isBullet = trimmed.startsWith('- ') || trimmed.startsWith('* ') || trimmed.startsWith('• ');
        const rawContent = isBullet ? trimmed.replace(/^[-*•]\s+/, '') : trimmed;

        // Numbered lists
        const numMatch = rawContent.match(/^(\d+)\.\s+(.*)/);
        const listPrefix = numMatch ? `${numMatch[1]}. ` : isBullet ? '• ' : '';
        const textContent = numMatch ? numMatch[2] : rawContent;

        // Parse Bold (**text**)
        const parts = textContent.split(/(\*\*.*?\*\*)/g);

        return (
          <div key={lineIdx} className={`text-xs ${isBullet || numMatch ? 'flex items-start gap-1.5 ml-1' : ''}`}>
            {(isBullet || numMatch) && (
              <span className="font-semibold text-purple-600 shrink-0">{listPrefix}</span>
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
                return <span key={partIdx}>{part}</span>;
              })}
            </div>
          </div>
        );
      })}
    </div>
  );
};

export const AIChatModal: React.FC<AIChatModalProps> = ({ isOpen, onClose }) => {
  const { t, i18n } = useTranslation();
  const { addToCart } = useCart();
  const { showToast } = useToast();
  const { currentUser } = useAuth();
  const { formatPrice } = useCurrency();

  const isEn = i18n.language === 'en';

  const INITIAL_MESSAGE: Message = {
    id: 'welcome',
    role: 'assistant',
    content: isEn
      ? 'Hi! 👋 I am **Tirenn AI Shopper**.\n\nHow can I help you today? You can ask for **gift recommendations**, search products by **price range**, or ask me to **add items to your shopping cart**!'
      : 'Halo! 👋 Saya **Tirenn AI Shopper**.\n\nAda yang bisa saya bantu carikan hari ini? Anda bisa meminta **rekomendasi hadiah**, mencari produk dengan **batas harga tertentu**, atau meminta saya **memasukkan barang ke keranjang**!',
  };

  const [messages, setMessages] = useState<Message[]>([INITIAL_MESSAGE]);
  const [input, setInput] = useState('');
  const [loading, setLoading] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement | null>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth' });
  };

  useEffect(() => {
    if (isOpen) {
      scrollToBottom();
    }
  }, [messages, isOpen]);

  const handleResetChat = () => {
    setMessages([INITIAL_MESSAGE]);
    setInput('');
    showToast(isEn ? 'Chat history cleared' : 'Riwayat chat berhasil dibersihkan', 'info');
  };

  const handleSend = async (textToSend?: string) => {
    const query = textToSend || input.trim();
    if (!query || loading) return;

    const userMsg: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: query,
    };

    setMessages((prev) => [...prev, userMsg]);
    if (!textToSend) setInput('');
    setLoading(true);

    try {
      const res = await fetch(`${AI_API_BASE_URL}/chat/shopper`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          messages: [...messages, userMsg].map((m) => ({
            role: m.role,
            content: m.content,
          })),
          is_authenticated: !!currentUser,
          user_name: currentUser?.name || undefined,
        }),
      });

      if (res.status === 429) {
        showToast(isEn ? 'Rate limit exceeded. Please wait a moment.' : 'Terlalu banyak permintaan. Mohon tunggu beberapa saat.', 'error');
        throw new Error('Rate limit exceeded');
      }

      if (!res.ok) throw new Error('Failed to connect to AI Shopper');
      const data = await res.json();

      const aiMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: data.reply || (isEn ? 'Here are the matching products:' : 'Berikut adalah hasil yang saya temukan:'),
        suggestedProducts: data.suggested_products || [],
        toolCalls: data.tool_calls || [],
      };

      setMessages((prev) => [...prev, aiMsg]);

      // If AI executed an add_to_cart action, dispatch to local CartContext (guest or user)
      const cartAction = data.cart_action;
      if (cartAction && cartAction.action === 'cart_added' && cartAction.product) {
        const prod = cartAction.product;
        addToCart(
          {
            id: prod.id,
            name: prod.name,
            sku: prod.sku,
            price: Number(prod.price || 0),
            image_url: prod.image_url || '',
            stock_quantity: Number(prod.stock_quantity || 99),
          } as any,
          prod.quantity || 1
        );
        showToast(
          `🛒 ${prod.name} (${prod.quantity || 1}x) ${isEn ? 'added to cart!' : 'berhasil dimasukkan ke keranjang!'}`,
          'success'
        );
      }
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: isEn 
            ? 'Sorry, I encountered an issue processing your request. Please try again shortly.' 
            : 'Maaf, saya sedang kesulitan memproses permintaan Anda. Silakan coba lagi sebentar lagi.',
        },
      ]);
    } finally {
      setLoading(false);
    }
  };

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 bg-slate-900/40 backdrop-blur-xs z-50 flex items-center justify-center p-4">
      <div className="bg-white rounded-2xl w-full max-w-lg h-[640px] flex flex-col shadow-2xl border border-slate-200 overflow-hidden animate-modal">
        {/* Header */}
        <div className="bg-slate-900 text-white p-4 flex items-center justify-between">
          <div className="flex items-center gap-2.5">
            <div className="w-8 h-8 rounded-full bg-purple-600 flex items-center justify-center text-sm shadow-inner">
              🤖
            </div>
            <div>
              <div className="font-bold text-sm leading-tight flex items-center gap-1.5">
                {t('ai_chat.title')}
                <span className="bg-purple-500/30 text-purple-200 text-[10px] px-1.5 py-0.2 rounded font-mono">
                  Qwen 2.5 1.5B
                </span>
              </div>
              <span className="text-[11px] text-slate-400">
                {t('ai_chat.subtitle')}
              </span>
            </div>
          </div>

          <div className="flex items-center gap-1">
            <button
              onClick={handleResetChat}
              title="Reset Chat"
              className="text-slate-400 hover:text-white p-1.5 rounded-lg hover:bg-slate-800 transition-colors text-xs cursor-pointer"
            >
              🔄
            </button>
            <button
              data-testid="ai-chat-close"
              onClick={onClose}
              className="text-slate-400 hover:text-white w-7 h-7 rounded-full hover:bg-slate-800 flex items-center justify-center cursor-pointer transition-colors text-sm"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Message Thread */}
        <div className="flex-1 p-4 overflow-y-auto space-y-3.5 bg-slate-50">
          {messages.map((m) => (
            <div
              key={m.id}
              className={`flex flex-col ${m.role === 'user' ? 'items-end' : 'items-start'}`}
            >
              <div
                className={`max-w-[85%] rounded-2xl px-4 py-2.5 shadow-xs ${
                  m.role === 'user'
                    ? 'bg-blue-600 text-white rounded-br-none'
                    : 'bg-white border border-slate-200 text-slate-800 rounded-bl-none'
                }`}
              >
                {m.role === 'user' ? (
                  <p className="text-xs leading-relaxed">{m.content}</p>
                ) : (
                  <FormattedMessageText text={m.content} />
                )}
              </div>

              {/* Tool Execution Logs Pill */}
              {m.toolCalls && m.toolCalls.length > 0 && (
                <div className="mt-1 flex flex-wrap gap-1">
                  {m.toolCalls.map((tc: any, idx: number) => (
                    <span
                      key={idx}
                      className="text-[10px] bg-purple-50 text-purple-700 border border-purple-200 px-2 py-0.5 rounded-full font-mono flex items-center gap-1"
                    >
                      <span>⚡ Tool:</span>
                      <b>{tc.name}</b>
                    </span>
                  ))}
                </div>
              )}

              {/* Visual Interactive Product Cards */}
              {m.suggestedProducts && m.suggestedProducts.length > 0 && (
                <div className="mt-2.5 w-full space-y-2">
                  <span className="text-[10px] font-bold uppercase tracking-wider text-slate-400 block px-1">
                    🛍️ {isEn ? 'Recommended Products' : 'Rekomendasi Produk Katalog'} ({m.suggestedProducts.length})
                  </span>
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-2">
                    {m.suggestedProducts.map((p: any) => (
                      <div
                        key={p.id}
                        data-testid={`chat-product-card-${p.id}`}
                        className="bg-white border border-slate-200 rounded-xl p-2.5 flex flex-col justify-between hover:border-purple-300 transition-all shadow-xs"
                      >
                        <div className="flex gap-2 items-start">
                          <img
                            src={p.image_url || 'https://images.unsplash.com/photo-1505740420928-5e560c06d30e?w=200&q=80'}
                            alt={p.name}
                            className="w-12 h-12 object-contain bg-slate-50 border border-slate-100 rounded-lg p-0.5 shrink-0"
                          />
                          <div className="min-w-0 flex-1">
                            <h4 className="font-semibold text-xs text-slate-900 leading-snug line-clamp-2">
                              {p.name}
                            </h4>
                            <span className="text-xs font-bold text-blue-600 block mt-0.5">
                              {formatPrice(p.price, p.currency || (p.sku?.startsWith('EN-') ? 'USD' : 'IDR'))}
                            </span>
                          </div>
                        </div>

                        <div className="mt-2 pt-2 border-t border-slate-100 flex items-center justify-between gap-1 text-[11px]">
                          <span className="text-slate-400 font-mono">
                            {p.stock_quantity > 0 ? `Stock: ${p.stock_quantity}` : 'Habis'}
                          </span>
                          <button
                            data-testid={`chat-add-to-cart-${p.id}`}
                            disabled={p.stock_quantity <= 0}
                            onClick={() => {
                              addToCart(
                                {
                                  id: p.id,
                                  name: p.name,
                                  sku: p.sku || `SKU-${p.id}`,
                                  price: Number(p.price),
                                  currency: p.currency || (p.sku?.startsWith('EN-') ? 'USD' : 'IDR'),
                                  image_url: p.image_url || '',
                                  stock_quantity: Number(p.stock_quantity || 10),
                                } as any,
                                1
                              );
                              showToast(
                                `🛒 ${p.name} ${isEn ? 'added to cart!' : 'berhasil dimasukkan ke keranjang!'}`,
                                'success'
                              );
                            }}
                            className={`px-2.5 py-1 rounded-md font-semibold transition-colors cursor-pointer text-[10px] ${
                              p.stock_quantity <= 0
                                ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                                : 'bg-purple-600 hover:bg-purple-700 text-white'
                            }`}
                          >
                            + {t('product.add_to_cart')}
                          </button>
                        </div>
                      </div>
                    ))}
                  </div>
                </div>
              )}
            </div>
          ))}

          {loading && (
            <div className="flex items-center gap-2 text-slate-400 text-xs py-2">
              <div className="w-4 h-4 border-2 border-purple-600 border-t-transparent rounded-full animate-spin" />
              <span>{t('ai_chat.thinking')}</span>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Quick Suggestion Chips */}
        <div className="px-3 py-2 bg-slate-100 border-t border-slate-200 flex gap-1.5 overflow-x-auto text-[11px]">
          <button
            onClick={() => handleSend(t('ai_chat.quick_1'))}
            className="shrink-0 bg-white hover:bg-slate-200 border border-slate-200 px-2.5 py-1 rounded-full text-slate-700 transition-colors cursor-pointer"
          >
            🎧 {t('ai_chat.quick_1')}
          </button>
          <button
            onClick={() => handleSend(t('ai_chat.quick_2'))}
            className="shrink-0 bg-white hover:bg-slate-200 border border-slate-200 px-2.5 py-1 rounded-full text-slate-700 transition-colors cursor-pointer"
          >
            👔 {t('ai_chat.quick_2')}
          </button>
          <button
            onClick={() => handleSend(t('ai_chat.quick_3'))}
            className="shrink-0 bg-white hover:bg-slate-200 border border-slate-200 px-2.5 py-1 rounded-full text-slate-700 transition-colors cursor-pointer"
          >
            ☕ {t('ai_chat.quick_3')}
          </button>
          <button
            onClick={() => handleSend(t('ai_chat.quick_4'))}
            className="shrink-0 bg-white hover:bg-slate-200 border border-slate-200 px-2.5 py-1 rounded-full text-slate-700 transition-colors cursor-pointer"
          >
            ✨ {t('ai_chat.quick_4')}
          </button>
        </div>

        {/* Input Bar */}
        <div className="p-3 bg-white border-t border-slate-200">
          <form
            onSubmit={(e) => {
              e.preventDefault();
              handleSend();
            }}
            className="flex items-center gap-2"
          >
            <input
              type="text"
              data-testid="ai-chat-input"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              placeholder={t('ai_chat.placeholder')}
              className="flex-1 bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2 text-xs text-slate-900 outline-none focus:bg-white focus:border-purple-600 transition-all"
            />
            <button
              type="submit"
              data-testid="ai-chat-send"
              disabled={loading || !input.trim()}
              className={`px-4 py-2 rounded-xl text-xs font-semibold transition-colors cursor-pointer ${
                loading || !input.trim()
                  ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                  : 'bg-purple-600 hover:bg-purple-700 text-white'
              }`}
            >
              {t('ai_chat.send')}
            </button>
          </form>
        </div>
      </div>
    </div>
  );
};
