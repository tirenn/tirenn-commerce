import React, { useState, useRef, useEffect } from 'react';
import { useCart } from '../context/CartContext';
import { useToast } from '../context/ToastContext';
import { formatRupiah } from '../utils/format';

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

        // Bullet point formatting
        let contentLine = line;
        let isBullet = false;
        if (trimmed.startsWith('- ') || trimmed.startsWith('* ') || trimmed.startsWith('• ')) {
          isBullet = true;
          contentLine = trimmed.replace(/^[-*•]\s+/, '');
        }

        // Numbered list formatting (e.g. 1. 2.)
        let listNumber = '';
        const matchNum = trimmed.match(/^(\d+)\.\s+(.*)$/);
        if (matchNum) {
          listNumber = matchNum[1];
          contentLine = matchNum[2];
        }

        // Parse **bold** and *italic* tokens
        const parts = contentLine.split(/(\*\*.*?\*\*|\*.*?\*)/g);
        const renderedParts = parts.map((part, partIdx) => {
          if (part.startsWith('**') && part.endsWith('**')) {
            return (
              <strong key={partIdx} className="font-bold text-slate-900">
                {part.slice(2, -2)}
              </strong>
            );
          }
          if (part.startsWith('*') && part.endsWith('*')) {
            return (
              <em key={partIdx} className="italic">
                {part.slice(1, -1)}
              </em>
            );
          }
          return part;
        });

        if (isBullet) {
          return (
            <div key={lineIdx} className="flex items-start gap-1.5 pl-2 text-slate-700">
              <span className="text-purple-600 font-bold text-xs leading-none mt-0.5">•</span>
              <span>{renderedParts}</span>
            </div>
          );
        }

        if (listNumber) {
          return (
            <div key={lineIdx} className="flex items-start gap-1.5 pl-1 font-semibold text-slate-900">
              <span className="text-purple-600">{listNumber}.</span>
              <span>{renderedParts}</span>
            </div>
          );
        }

        return <p key={lineIdx}>{renderedParts}</p>;
      })}
    </div>
  );
};

export const AIChatModal: React.FC<AIChatModalProps> = ({ isOpen, onClose }) => {
  const { addToCart } = useCart();
  const { showToast } = useToast();

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
      const res = await fetch(`http://localhost:8000/api/v1/chat/shopper`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          messages: [...messages, userMsg].map((m) => ({
            role: m.role,
            content: m.content,
          })),
        }),
      });

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

      // If AI executed an add_to_cart action
      if (data.cart_action && (data.cart_action.product_id || data.cart_action.sku)) {
        showToast(
          `Produk ${data.cart_action.sku || data.cart_action.name} dimasukkan ke keranjang!`,
          'success'
        );
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
              </div>
              <div className="text-[11px] text-slate-300">Asisten Belanja Pintar dengan Tool Calling</div>
            </div>
          </div>

          <div className="flex items-center gap-2">
            <button
              onClick={handleResetChat}
              title="Bersihkan riwayat percakapan"
              className="text-slate-300 hover:text-white text-[11px] px-2.5 py-1 rounded-lg flex items-center gap-1 bg-slate-800 hover:bg-slate-700 transition-colors cursor-pointer border border-slate-700 font-medium"
            >
              <span>🔄</span>
              <span className="hidden sm:inline">Reset Chat</span>
            </button>
            <button
              onClick={onClose}
              className="text-slate-400 hover:text-white text-lg w-7 h-7 rounded-lg flex items-center justify-center hover:bg-slate-800 transition-colors cursor-pointer"
            >
              ✕
            </button>
          </div>
        </div>

        {/* Chat History */}
        <div className="flex-1 overflow-y-auto p-4 space-y-4 bg-slate-50 text-xs">
          {messages.map((m) => {
            const isUser = m.role === 'user';
            return (
              <div key={m.id} className={`flex flex-col ${isUser ? 'items-end' : 'items-start'}`}>
                {/* Tool Calling Badges */}
                {m.toolCalls && m.toolCalls.length > 0 && (
                  <div className="mb-1.5 flex flex-wrap gap-1">
                    {m.toolCalls.map((tc, idx) => {
                      const paramLabel =
                        tc.args?.product_name_or_query ||
                        tc.args?.query ||
                        (tc.args?.sku ? `SKU: ${tc.args.sku}` : '');
                      return (
                        <span
                          key={idx}
                          className="inline-flex items-center gap-1 bg-purple-50 text-purple-800 text-[10px] px-2 py-0.5 rounded font-mono border border-purple-200"
                        >
                          ⚡ Tool: {tc.tool}({paramLabel ? `"${paramLabel}"` : ''})
                        </span>
                      );
                    })}
                  </div>
                )}

                {/* Message Bubble */}
                <div
                  className={`max-w-[88%] rounded-2xl px-4 py-3 shadow-xs leading-relaxed ${
                    isUser
                      ? 'bg-blue-600 text-white rounded-br-none whitespace-pre-wrap'
                      : 'bg-white text-slate-800 border border-slate-200 rounded-bl-none'
                  }`}
                >
                  {isUser ? m.content : <FormattedMessageText text={m.content} />}
                </div>

                {/* Suggested Product Cards with Pictures & SKU */}
                {m.suggestedProducts && m.suggestedProducts.length > 0 && (
                  <div className="mt-2.5 w-full space-y-2">
                    <div className="text-[11px] font-bold text-slate-500 flex items-center gap-1">
                      <span>🛍️ Produk Terkait:</span>
                    </div>
                    <div className="grid grid-cols-1 sm:grid-cols-2 gap-2.5">
                      {m.suggestedProducts.map((p) => (
                        <div
                          key={p.id || p.sku}
                          className="bg-white border border-slate-200 rounded-xl p-2.5 flex flex-col justify-between shadow-xs hover:border-purple-400 transition-colors group"
                        >
                          {/* Product Image */}
                          <div className="relative w-full h-28 bg-slate-100 rounded-lg overflow-hidden mb-2">
                            {p.image_url ? (
                              <img
                                src={p.image_url}
                                alt={p.name}
                                className="w-full h-full object-cover group-hover:scale-105 transition-transform duration-300"
                                onError={(e) => {
                                  (e.target as HTMLImageElement).src =
                                    'https://images.unsplash.com/photo-1523275335684-37898b6baf30?w=600&auto=format&fit=crop&q=80';
                                }}
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
            placeholder="Tanyakan apapun ke AI Shopper..."
            className="flex-1 bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2 text-xs text-slate-900 outline-none focus:border-purple-600 focus:bg-white transition-all"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            disabled={loading}
          />
          <button
            type="submit"
            disabled={!input.trim() || loading}
            className={`px-4 py-2 rounded-xl text-xs font-bold transition-colors cursor-pointer ${
              !input.trim() || loading
                ? 'bg-slate-100 text-slate-400 cursor-not-allowed'
                : 'bg-purple-600 hover:bg-purple-700 text-white shadow-xs'
            }`}
          >
            Kirim
          </button>
        </form>
      </div>
    </div>
  );
};
