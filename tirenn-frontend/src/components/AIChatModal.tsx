import React, { useState, useRef, useEffect } from 'react';
import { useCart } from '../context/CartContext';
import { useToast } from '../context/ToastContext';
import { useAuth } from '../context/AuthContext';
import { formatRupiah } from '../utils/format';
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

// Lightweight Markdown text renderer (handles **bold**, *italic*, list items, clean bullets, and strips redundant image URLs)
const FormattedMessageText: React.FC<{ text: string }> = ({ text }) => {
  const lines = text.split('\n');

  return (
    <div className="space-y-1.5 leading-relaxed">
      {lines.map((line, lineIdx) => {
        const trimmed = line.trim();
        if (!trimmed) return <div key={lineIdx} className="h-1" />;

        // Clean out raw image references or Markdown image embeds (![...] or [Lihat Gambar])
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

export const AIChatModal: React.FC<AIChatModalProps> = ({ isOpen, onClose, onAuthRequired }) => {
  const { addToCart } = useCart();
  const { showToast } = useToast();
  const { currentUser } = useAuth();

  const INITIAL_MESSAGE: Message = {
    id: 'welcome',
    role: 'assistant',
    content: 'Halo! 👋 Saya **Tirenn AI Shopper**.\n\nAda yang bisa saya bantu carikan hari ini? Anda bisa meminta **rekomendasi hadiah**, mencari produk dengan **batas harga tertentu**, atau menanyakan **ketersediaan stok barang**!',
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
    showToast('Riwayat chat berhasil dibersihkan', 'info');
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
        showToast('Terlalu banyak permintaan (Rate limit exceeded). Mohon tunggu beberapa saat.', 'error');
        throw new Error('Rate limit exceeded');
      }

      if (!res.ok) throw new Error('Gagal menghubungi AI Shopper');
      const data = await res.json();

      const aiMsg: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: data.reply || 'Berikut adalah hasil yang saya temukan:',
        suggestedProducts: data.suggested_products || [],
        toolCalls: data.tool_calls || [],
      };

      setMessages((prev) => [...prev, aiMsg]);

      // If AI executed an add_to_cart action or returned auth_required
      const cartPayload: any = data.cart_action || (
        Array.isArray(data.tool_calls) 
          ? data.tool_calls.find((tc: any) => tc.name === 'add_to_cart' && tc.output)?.output 
          : null
      );

      if (cartPayload) {
        if (cartPayload.action === 'auth_required' || !currentUser) {
          showToast('🔒 Silakan login terlebih dahulu untuk memasukkan produk ke keranjang belanja.', 'info');
          if (onAuthRequired) {
            onAuthRequired();
          }
        } else if (cartPayload.action === 'cart_added' && (cartPayload.product_id || cartPayload.id || cartPayload.sku)) {
          addToCart(
            {
              id: cartPayload.product_id || cartPayload.id,
              name: cartPayload.name,
              sku: cartPayload.sku,
              price: Number(cartPayload.price || 0),
              image_url: cartPayload.image_url || '',
              stock_quantity: Number(cartPayload.stock_quantity || 99),
            } as any,
            cartPayload.quantity || 1
          );
          showToast(
            `🛒 ${cartPayload.name} (${cartPayload.quantity || 1}x) berhasil dimasukkan ke keranjang!`,
            'success'
          );
        }
      }
    } catch (err: any) {
      setMessages((prev) => [
        ...prev,
        {
          id: (Date.now() + 1).toString(),
          role: 'assistant',
          content: 'Maaf, saya sedang kesulitan memproses permintaan Anda. Silakan coba lagi sebentar lagi.',
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
                Tirenn AI Shopper
                <span className="bg-purple-500/30 text-purple-200 text-[10px] px-1.5 py-0.2 rounded font-mono">
                  Qwen 2.5
                </span>
                {currentUser && (
                  <span className="bg-emerald-500/30 text-emerald-300 text-[10px] px-1.5 py-0.2 rounded font-medium">
                    {currentUser.name}
                  </span>
                )}
              </div>
              <div className="text-[11px] text-slate-300">Asisten Belanja Pintar dengan Tool Calling</div>
            </div>
          </div>
          <div className="flex items-center gap-1.5">
            {/* Reset Chat Button */}
            <button
              onClick={handleResetChat}
              title="Bersihkan riwayat percakapan"
              className="text-slate-400 hover:text-white p-1.5 hover:bg-slate-800 rounded-lg transition-colors cursor-pointer text-xs flex items-center gap-1"
            >
              <span>🔄</span>
              <span className="hidden sm:inline text-[11px]">Reset</span>
            </button>
            <button
              onClick={onClose}
              className="text-slate-400 hover:text-white p-1.5 hover:bg-slate-800 rounded-lg transition-colors cursor-pointer text-lg leading-none"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Messages Body */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 bg-slate-50/50">
          {messages.map((m) => {
            const isUser = m.role === 'user';
            return (
              <div key={m.id} className={`flex flex-col ${isUser ? 'items-end' : 'items-start'}`}>
                <div
                  className={`max-w-[85%] rounded-2xl p-3.5 shadow-xs ${
                    isUser
                      ? 'bg-purple-600 text-white rounded-br-xs text-xs'
                      : 'bg-white text-slate-800 border border-slate-200/80 rounded-bl-xs'
                  }`}
                >
                  {isUser ? (
                    <div className="whitespace-pre-wrap">{m.content}</div>
                  ) : (
                    <FormattedMessageText text={m.content} />
                  )}
                </div>

                {/* Render Tool Calling Badges */}
                {!isUser && m.toolCalls && m.toolCalls.length > 0 && (
                  <div className="mt-1.5 flex flex-wrap gap-1">
                    {m.toolCalls.map((tc, idx) => (
                      <span
                        key={idx}
                        className="inline-flex items-center gap-1 text-[10px] font-mono bg-purple-100 text-purple-800 px-2 py-0.5 rounded-md border border-purple-200"
                      >
                        ⚡ Tool: {tc.name}
                      </span>
                    ))}
                  </div>
                )}

                {/* Render Suggested Products Carousel / Cards */}
                {!isUser && m.suggestedProducts && m.suggestedProducts.length > 0 && (
                  <div className="mt-3 w-full max-w-[95%]">
                    <div className="text-[11px] font-semibold text-slate-500 mb-2 flex items-center gap-1">
                      <span>🛍️ Produk Terkait:</span>
                    </div>
                    <div className="grid grid-cols-2 gap-2">
                      {m.suggestedProducts.map((p) => (
                        <div
                          key={p.id}
                          className="bg-white border border-slate-200 rounded-xl p-2.5 shadow-xs flex flex-col justify-between hover:border-purple-300 transition-colors"
                        >
                          {/* Image Thumbnail */}
                          <div className="w-full h-24 bg-slate-100 rounded-lg mb-2 overflow-hidden relative">
                            {p.image_url ? (
                              <img
                                src={p.image_url}
                                alt={p.name}
                                className="w-full h-full object-cover"
                              />
                            ) : (
                              <div className="w-full h-full flex items-center justify-center text-slate-400 text-xl">
                                📦
                              </div>
                            )}
                            {p.sku && (
                              <span className="absolute top-1.5 left-1.5 bg-slate-900/80 backdrop-blur-xs text-white text-[9px] font-mono px-1.5 py-0.5 rounded">
                                {p.sku}
                              </span>
                            )}
                          </div>

                          {/* Product Details */}
                          <div className="space-y-1 mb-2.5">
                            <div className="font-semibold text-slate-900 line-clamp-2 text-[11px] leading-tight">
                              {p.name}
                            </div>
                            <div className="text-purple-700 font-bold text-xs">
                              {formatRupiah(p.price)}
                            </div>
                          </div>

                          {/* Add to Cart Button */}
                          <button
                            onClick={() => {
                              if (!currentUser) {
                                showToast(
                                  '🔒 Silakan login terlebih dahulu untuk memasukkan produk ke keranjang.',
                                  'info'
                                );
                                if (onAuthRequired) onAuthRequired();
                                return;
                              }
                              addToCart(p as any);
                              showToast(
                                `${p.sku || p.name} berhasil ditambahkan ke keranjang!`,
                                'success'
                              );
                            }}
                            className="w-full bg-slate-900 hover:bg-purple-600 text-white text-[11px] font-semibold py-1.5 px-2 rounded-lg transition-colors cursor-pointer text-center flex items-center justify-center gap-1 shadow-xs"
                          >
                            <span>+ Masukkan Keranjang</span>
                          </button>
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            );
          })}

          {loading && (
            <div className="flex items-center gap-2 text-slate-500 text-xs bg-white border border-slate-200 p-3 rounded-2xl w-fit shadow-xs animate-pulse">
              <span className="w-3.5 h-3.5 border-2 border-purple-600 border-t-transparent rounded-full animate-spin"></span>
              <span>AI Shopper sedang mencari & menjalankan tool...</span>
            </div>
          )}

          <div ref={messagesEndRef} />
        </div>

        {/* Suggestion Chips */}
        <div className="px-4 py-2 bg-white border-t border-slate-100 flex items-center gap-1.5 overflow-x-auto text-[11px] no-scrollbar">
          <button
            onClick={() => handleSend('Cari pakaian wanita di bawah Rp 300.000')}
            className="whitespace-nowrap bg-slate-100 hover:bg-purple-50 hover:text-purple-700 text-slate-700 px-2.5 py-1 rounded-full cursor-pointer transition-colors"
          >
            👗 Pakaian wanita &lt; 300rb
          </button>
          <button
            onClick={() => handleSend('Cari celana jeans pria di bawah Rp 300.000')}
            className="whitespace-nowrap bg-slate-100 hover:bg-purple-50 hover:text-purple-700 text-slate-700 px-2.5 py-1 rounded-full cursor-pointer transition-colors"
          >
            👖 Celana jeans &lt; 300rb
          </button>
          <button
            onClick={() => handleSend('Rekomendasi biji kopi arabika specialty')}
            className="whitespace-nowrap bg-slate-100 hover:bg-purple-50 hover:text-purple-700 text-slate-700 px-2.5 py-1 rounded-full cursor-pointer transition-colors"
          >
            ☕ Kopi Arabika
          </button>
        </div>

        {/* Input Bar */}
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleSend();
          }}
          className="p-3 bg-white border-t border-slate-200 flex items-center gap-2"
        >
          <input
            type="text"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Tanyakan rekomendasi, harga, atau stok..."
            className="flex-1 border border-slate-300 rounded-xl px-3.5 py-2 text-xs focus:outline-none focus:border-purple-600 focus:ring-1 focus:ring-purple-600"
          />
          <button
            type="submit"
            disabled={!input.trim() || loading}
            className="bg-purple-600 hover:bg-purple-700 disabled:opacity-50 text-white px-4 py-2 rounded-xl text-xs font-semibold cursor-pointer transition-colors"
          >
            Kirim
          </button>
        </form>
      </div>
    </div>
  );
};
