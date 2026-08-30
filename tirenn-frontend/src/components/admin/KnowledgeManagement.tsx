import React, { useState, useEffect, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { useAuth } from '../../context/AuthContext';
import { useToast } from '../../context/ToastContext';
import type { KnowledgeDocument, KnowledgeChunkResult } from '../../types';

const AI_API_BASE_URL = import.meta.env.VITE_AI_SERVICE_URL || 'http://localhost:8000/api/v1';

export const KnowledgeManagement: React.FC = () => {
  const { t } = useTranslation();
  const { token } = useAuth();
  const { showToast } = useToast();

  const [documents, setDocuments] = useState<KnowledgeDocument[]>([]);
  const [loadingDocs, setLoadingDocs] = useState(true);

  // Upload States
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [docTitle, setDocTitle] = useState('');
  const [docType, setDocType] = useState('SOP_CUSTOMER');
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState('');
  const fileInputRef = useRef<HTMLInputElement | null>(null);

  // RAG Search Playground States
  const [testQuery, setTestQuery] = useState('');
  const [ragScope, setRagScope] = useState('ALL');
  const [searchingRAG, setSearchingRAG] = useState(false);
  const [ragResults, setRagResults] = useState<KnowledgeChunkResult[]>([]);

  // Load indexed documents from AI service with JWT validation
  const loadDocuments = async () => {
    try {
      setLoadingDocs(true);
      const res = await fetch(`${AI_API_BASE_URL}/knowledge/documents`, {
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      if (res.ok) {
        const data = await res.json();
        if (data && Array.isArray(data.documents)) {
          setDocuments(data.documents);
        }
      } else {
        console.error('Failed to load knowledge documents', await res.text());
      }
    } catch (err) {
      console.error('Error fetching knowledge documents', err);
    } finally {
      setLoadingDocs(false);
    }
  };

  useEffect(() => {
    loadDocuments();
  }, [token]);

  // Handle PDF File Selection
  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      const file = e.target.files[0];
      if (!file.name.toLowerCase().endsWith('.pdf')) {
        showToast(t('knowledge.err_pdf_only'), 'error');
        return;
      }
      setSelectedFile(file);
      if (!docTitle) {
        setDocTitle(file.name.replace(/\.[^/.]+$/, '').replace(/_/g, ' '));
      }
    }
  };

  // Upload PDF & Index In-Memory to pgvector
  const handleUploadAndIndex = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!selectedFile) {
      showToast(t('knowledge.err_select_first'), 'error');
      return;
    }

    try {
      setUploading(true);
      setUploadProgress(t('knowledge.indexing_btn'));

      const formData = new FormData();
      formData.append('file', selectedFile);
      if (docTitle) formData.append('title', docTitle);
      formData.append('doc_type', docType);

      const res = await fetch(`${AI_API_BASE_URL}/knowledge/upload-pdf`, {
        method: 'POST',
        headers: {
          'Authorization': `Bearer ${token}`
        },
        body: formData
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.detail || 'Failed to upload and vectorize PDF');
      }

      const data = await res.json();
      showToast(
        t('knowledge.upload_success', {
          title: data.document?.title || selectedFile.name,
          chunks: data.document?.total_chunks || 0
        }),
        'success'
      );

      // Reset upload form
      setSelectedFile(null);
      setDocTitle('');
      if (fileInputRef.current) fileInputRef.current.value = '';
      loadDocuments();
    } catch (err: any) {
      showToast(`⚠️ ${err.message || 'Error uploading PDF'}`, 'error');
    } finally {
      setUploading(false);
      setUploadProgress('');
    }
  };

  // Delete Document
  const handleDeleteDocument = async (docId: number, title: string) => {
    if (!window.confirm(t('knowledge.delete_confirm', { title }))) {
      return;
    }

    try {
      const res = await fetch(`${AI_API_BASE_URL}/knowledge/documents/${docId}`, {
        method: 'DELETE',
        headers: {
          'Authorization': `Bearer ${token}`
        }
      });
      if (res.ok) {
        showToast(t('knowledge.delete_success'), 'info');
        loadDocuments();
      } else {
        throw new Error('Failed to delete document');
      }
    } catch (err: any) {
      showToast(err.message, 'error');
    }
  };

  // Test RAG Vector Search Playground
  const handleTestRAG = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!testQuery.trim()) return;

    try {
      setSearchingRAG(true);
      const res = await fetch(`${AI_API_BASE_URL}/knowledge/query`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          query: testQuery,
          limit: 4,
          score_threshold: 0.10,
          doc_type: ragScope === 'ALL' ? undefined : ragScope
        })
      });
      if (res.ok) {
        const data = await res.json();
        setRagResults(data.results || []);
      }
    } catch (err) {
      console.error('RAG search error', err);
    } finally {
      setSearchingRAG(false);
    }
  };

  return (
    <div className="space-y-8">
      {/* Top Banner Header */}
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <h2 className="text-xl sm:text-2xl font-black text-slate-900 tracking-tight flex items-center gap-2.5">
            <span className="text-2xl">📚</span>
            {t('knowledge.title')}
          </h2>
          <p className="text-xs sm:text-sm text-slate-500 mt-1">
            {t('knowledge.subtitle')}
          </p>
        </div>
        <button
          onClick={loadDocuments}
          className="px-3.5 py-2 text-xs font-semibold text-slate-700 bg-white hover:bg-slate-50 border border-slate-200 rounded-xl transition-all shadow-xs flex items-center gap-1.5 cursor-pointer"
        >
          <span>🔄</span> {t('knowledge.refresh')}
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
        
        {/* Left Column (Upload Box): 5 cols */}
        <div className="lg:col-span-5 space-y-6">
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-xs">
            <div className="flex items-center gap-2 text-sm font-bold text-slate-900 border-b border-slate-100 pb-3 mb-5">
              <span className="p-1.5 bg-purple-50 text-purple-600 rounded-lg text-xs">📤</span>
              {t('knowledge.upload_title')}
            </div>

            <form onSubmit={handleUploadAndIndex} className="space-y-4">
              
              {/* Drag & Drop File Selector */}
              <div>
                <label className="block text-xs font-bold text-slate-700 uppercase tracking-wider mb-2">
                  {t('knowledge.select_pdf')}
                </label>
                <div
                  onClick={() => fileInputRef.current?.click()}
                  className={`border-2 border-dashed rounded-xl p-6 text-center cursor-pointer transition-all ${
                    selectedFile
                      ? 'border-purple-500 bg-purple-50/50'
                      : 'border-slate-300 hover:border-purple-400 bg-slate-50 hover:bg-slate-50/80'
                  }`}
                >
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".pdf,application/pdf"
                    onChange={handleFileChange}
                    className="hidden"
                  />
                  {selectedFile ? (
                    <div>
                      <span className="text-3xl block mb-2">📄</span>
                      <p className="font-bold text-xs text-purple-900 break-all">{selectedFile.name}</p>
                      <p className="text-[11px] text-purple-600 mt-1">
                        {(selectedFile.size / 1024).toFixed(1)} KB • {t('knowledge.ready_to_index')}
                      </p>
                    </div>
                  ) : (
                    <div>
                      <span className="text-3xl block mb-2 text-slate-400">📑</span>
                      <p className="font-bold text-xs text-slate-700">
                        {t('knowledge.click_or_drag')}
                      </p>
                      <p className="text-[11px] text-slate-400 mt-1">
                        {t('knowledge.in_memory_note')}
                      </p>
                    </div>
                  )}
                </div>
              </div>

              {/* Document Title */}
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t('knowledge.doc_title_label')}
                </label>
                <input
                  type="text"
                  value={docTitle}
                  onChange={(e) => setDocTitle(e.target.value)}
                  placeholder={t('knowledge.doc_title_placeholder')}
                  className="w-full text-xs bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 outline-none focus:bg-white focus:border-purple-600 font-medium transition-all"
                />
              </div>

              {/* Document Category / Target Audience */}
              <div>
                <label className="block text-xs font-bold text-slate-700 mb-1">
                  {t('knowledge.scope_label')}
                </label>
                <select
                  value={docType}
                  onChange={(e) => setDocType(e.target.value)}
                  className="w-full text-xs bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 outline-none focus:bg-white focus:border-purple-600 font-medium transition-all"
                >
                  <option value="SOP_CUSTOMER">{t('knowledge.scope_customer_option')}</option>
                  <option value="SOP_ADMIN">{t('knowledge.scope_admin_option')}</option>
                </select>
                <p className="text-[10px] text-slate-500 mt-1">
                  {docType === 'SOP_CUSTOMER'
                    ? t('knowledge.scope_customer_hint')
                    : t('knowledge.scope_admin_hint')}
                </p>
              </div>

              {/* Upload Button */}
              <button
                type="submit"
                disabled={!selectedFile || uploading}
                className="w-full py-3 px-4 bg-purple-700 hover:bg-purple-800 disabled:bg-slate-200 disabled:text-slate-400 text-white font-bold text-xs rounded-xl transition-all shadow-sm flex items-center justify-center gap-2 cursor-pointer"
              >
                {uploading ? (
                  <>
                    <span className="animate-spin text-sm">⚡</span>
                    <span>{uploadProgress || t('knowledge.indexing_btn')}</span>
                  </>
                ) : (
                  <>
                    <span>⚡</span>
                    <span>{t('knowledge.index_btn')}</span>
                  </>
                )}
              </button>
            </form>
          </div>
        </div>

        {/* Right Column (Document List & RAG Playground): 7 cols */}
        <div className="lg:col-span-7 space-y-6">
          
          {/* Indexed Documents Table */}
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-xs">
            <div className="flex items-center justify-between border-b border-slate-100 pb-3 mb-4">
              <div className="flex items-center gap-2 text-sm font-bold text-slate-900">
                <span className="p-1.5 bg-blue-50 text-blue-600 rounded-lg text-xs">📑</span>
                {t('knowledge.indexed_docs_title')} ({documents.length})
              </div>
            </div>

            {loadingDocs ? (
              <div className="py-12 text-center text-xs text-slate-400 animate-pulse">
                Loading...
              </div>
            ) : documents.length === 0 ? (
              <div className="py-12 text-center text-xs text-slate-400 bg-slate-50 border border-dashed border-slate-200 rounded-xl p-6">
                <span className="text-2xl block mb-2">📂</span>
                {t('knowledge.no_docs')}
              </div>
            ) : (
              <div className="overflow-x-auto">
                <table className="w-full text-left text-xs">
                  <thead>
                    <tr className="border-b border-slate-200 text-slate-500 font-bold uppercase tracking-wider text-[10px]">
                      <th className="py-2.5 px-3">{t('knowledge.col_title')}</th>
                      <th className="py-2.5 px-3">{t('knowledge.col_scope')}</th>
                      <th className="py-2.5 px-3 text-center">{t('knowledge.col_pages')}</th>
                      <th className="py-2.5 px-3 text-center">{t('knowledge.col_chunks')}</th>
                      <th className="py-2.5 px-3 text-right">{t('knowledge.col_action')}</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-slate-100">
                    {documents.map((doc) => (
                      <tr key={doc.id} className="hover:bg-slate-50/80 transition-colors">
                        <td className="py-3 px-3">
                          <div className="font-bold text-slate-900 leading-snug">{doc.title}</div>
                          <div className="text-[10px] text-slate-400 font-mono mt-0.5">{doc.filename}</div>
                        </td>
                        <td className="py-3 px-3">
                          <span className={`inline-flex items-center gap-1 px-2.5 py-0.5 rounded-full font-mono text-[10px] font-bold ${
                            doc.doc_type === 'SOP_CUSTOMER'
                              ? 'bg-emerald-50 text-emerald-700 border border-emerald-200'
                              : 'bg-amber-50 text-amber-700 border border-amber-200'
                          }`}>
                            {doc.doc_type === 'SOP_CUSTOMER' ? t('knowledge.badge_customer') : t('knowledge.badge_admin')}
                          </span>
                        </td>
                        <td className="py-3 px-3 text-center font-mono text-slate-600">
                          {doc.total_pages}
                        </td>
                        <td className="py-3 px-3 text-center font-bold text-purple-700 font-mono">
                          {doc.total_chunks}
                        </td>
                        <td className="py-3 px-3 text-right">
                          <button
                            onClick={() => handleDeleteDocument(doc.id, doc.title)}
                            className="text-slate-400 hover:text-rose-600 transition-colors p-1 rounded-md hover:bg-rose-50 cursor-pointer"
                            title={t('admin.delete')}
                          >
                            🗑️
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            )}
          </div>

          {/* RAG Interactive Semantic Playground */}
          <div className="bg-white border border-slate-200 rounded-2xl p-6 shadow-xs">
            <div className="flex items-center justify-between gap-2 text-sm font-bold text-slate-900 border-b border-slate-100 pb-3 mb-4">
              <div className="flex items-center gap-2">
                <span className="p-1.5 bg-emerald-50 text-emerald-600 rounded-lg text-xs">🎯</span>
                {t('knowledge.playground_title')}
              </div>
              <div className="flex items-center gap-1.5 text-xs font-normal">
                <span className="text-[11px] text-slate-400 font-semibold">{t('knowledge.scope_filter_label')}</span>
                <select
                  value={ragScope}
                  onChange={(e) => setRagScope(e.target.value)}
                  className="text-xs bg-slate-50 border border-slate-200 rounded-lg px-2 py-1 outline-none font-medium focus:border-emerald-600"
                >
                  <option value="ALL">{t('knowledge.scope_all')}</option>
                  <option value="SOP_CUSTOMER">{t('knowledge.scope_customer')}</option>
                  <option value="SOP_ADMIN">{t('knowledge.scope_admin')}</option>
                </select>
              </div>
            </div>

            <form onSubmit={handleTestRAG} className="flex gap-2 mb-4">
              <input
                type="text"
                value={testQuery}
                onChange={(e) => setTestQuery(e.target.value)}
                placeholder={t('knowledge.search_placeholder')}
                className="flex-1 text-xs bg-slate-50 border border-slate-200 rounded-xl px-3.5 py-2.5 outline-none focus:bg-white focus:border-emerald-600 font-medium transition-all"
              />
              <button
                type="submit"
                disabled={searchingRAG || !testQuery.trim()}
                className="px-4 py-2.5 bg-emerald-600 hover:bg-emerald-700 disabled:bg-slate-200 text-white font-bold text-xs rounded-xl transition-all shadow-xs flex items-center gap-1.5 cursor-pointer"
              >
                {searchingRAG ? <span className="animate-spin">⚡</span> : <span>🔍</span>}
                <span>{t('knowledge.test_btn')}</span>
              </button>
            </form>

            {/* Retrieved Chunks Display */}
            {ragResults.length > 0 && (
              <div className="space-y-3 mt-4 pt-4 border-t border-slate-100">
                <span className="text-[11px] font-bold uppercase tracking-wider text-slate-400 block">
                  {t('knowledge.retrieved_chunks', { count: ragResults.length })}
                </span>
                <div className="space-y-2.5">
                  {ragResults.map((chunk, idx) => (
                    <div
                      key={idx}
                      className="bg-slate-50 border border-slate-200 rounded-xl p-3.5 text-xs text-slate-700 space-y-1.5"
                    >
                      <div className="flex items-center justify-between text-[11px]">
                        <span className="font-bold text-slate-900 flex items-center gap-1">
                          <span>📄</span> {chunk.document_title} <span className="text-slate-400">(Hal {chunk.page_number})</span>
                        </span>
                        <span className="font-mono font-bold text-emerald-700 bg-emerald-50 px-2 py-0.5 rounded-full border border-emerald-200 text-[10px]">
                          {t('knowledge.similarity', { score: (chunk.score * 100).toFixed(1) })}
                        </span>
                      </div>
                      <p className="text-slate-600 text-[11px] leading-relaxed whitespace-pre-line bg-white p-2.5 rounded-lg border border-slate-100 font-sans">
                        {chunk.content}
                      </p>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>

        </div>

      </div>
    </div>
  );
};
