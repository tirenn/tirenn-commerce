import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import idTranslations from './locales/id.json';
import enTranslations from './locales/en.json';

const savedLang = localStorage.getItem('tirenn_lang') || 'id';

i18n
  .use(initReactI18next)
  .init({
    resources: {
      id: { translation: idTranslations },
      en: { translation: enTranslations }
    },
    lng: savedLang,
    fallbackLng: 'id',
    interpolation: {
      escapeValue: false
    }
  });

export default i18n;
