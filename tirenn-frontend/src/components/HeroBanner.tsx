import React from 'react';
import { useTranslation } from 'react-i18next';

interface HeroBannerProps {
  onExplore?: () => void;
  onOpenAI?: () => void;
}

export const HeroBanner: React.FC<HeroBannerProps> = ({ onExplore, onOpenAI }) => {
  const { t } = useTranslation();

  return (
    <div className="mb-6 bg-linear-to-r from-slate-900 via-indigo-950 to-slate-900 text-white rounded-2xl p-6 sm:p-8 flex flex-col md:flex-row md:items-center justify-between gap-6 shadow-xl border border-slate-800 relative overflow-hidden">
      {/* Background glow effects */}
      <div className="absolute top-0 right-0 -mt-12 -mr-12 w-64 h-64 bg-blue-500/10 rounded-full blur-3xl pointer-events-none" />
      <div className="absolute bottom-0 left-1/3 -mb-12 w-64 h-64 bg-purple-500/10 rounded-full blur-3xl pointer-events-none" />

      <div className="relative z-10 max-w-xl">
        <div className="inline-flex items-center gap-1.5 px-2.5 py-0.5 rounded-full bg-purple-500/20 text-purple-300 text-[11px] font-medium mb-3 border border-purple-500/30">
          <span>✨</span>
          <span>{t('hero.badge')}</span>
        </div>

        <h1 className="text-2xl sm:text-3xl lg:text-4xl font-extrabold tracking-tight text-white mb-2 leading-tight">
          {t('hero.title_part1')}{' '}
          <span className="text-transparent bg-clip-text bg-linear-to-r from-blue-400 to-purple-400">
            {t('hero.title_part2')}
          </span>
        </h1>

        <p className="text-slate-300 text-xs sm:text-sm leading-relaxed mb-5">
          {t('hero.subtitle')}
        </p>

        <div className="flex flex-wrap items-center gap-3">
          {onExplore && (
            <button
              onClick={onExplore}
              className="bg-blue-600 hover:bg-blue-500 text-white text-xs font-semibold px-4 py-2.5 rounded-xl transition-all shadow-md cursor-pointer flex items-center gap-1.5"
            >
              <span>🛍️</span>
              <span>{t('hero.cta_explore')}</span>
            </button>
          )}

          {onOpenAI && (
            <button
              onClick={onOpenAI}
              className="bg-purple-600/80 hover:bg-purple-600 text-white text-xs font-semibold px-4 py-2.5 rounded-xl transition-all border border-purple-400/30 shadow-md cursor-pointer flex items-center gap-1.5"
            >
              <span>🤖</span>
              <span>{t('hero.cta_ai')}</span>
            </button>
          )}
        </div>
      </div>

      {/* Hero Stats */}
      <div className="relative z-10 grid grid-cols-3 gap-3 sm:gap-4 border-t md:border-t-0 md:border-l border-slate-800/80 pt-4 md:pt-0 md:pl-6 text-center">
        <div className="bg-slate-800/40 p-3 rounded-xl border border-slate-700/50">
          <div className="text-lg sm:text-xl font-bold text-white">280+</div>
          <div className="text-[10px] text-slate-400 uppercase tracking-wider mt-0.5">{t('hero.stat_products')}</div>
        </div>
        <div className="bg-slate-800/40 p-3 rounded-xl border border-slate-700/50">
          <div className="text-lg sm:text-xl font-bold text-purple-400">~5s</div>
          <div className="text-[10px] text-slate-400 uppercase tracking-wider mt-0.5">{t('hero.stat_speed')}</div>
        </div>
        <div className="bg-slate-800/40 p-3 rounded-xl border border-slate-700/50">
          <div className="text-lg sm:text-xl font-bold text-emerald-400">100%</div>
          <div className="text-[10px] text-slate-400 uppercase tracking-wider mt-0.5">{t('hero.stat_satisfaction')}</div>
        </div>
      </div>
    </div>
  );
};
