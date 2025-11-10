import { defineConfig, presetUno } from 'unocss'

export default defineConfig({
  presets: [
    presetUno(),
  ],
  theme: {
    colors: {
      // Abfall-Farben
      'restmuell': '#495057',
      'biotonne': '#28a745',
      'papiertonne': '#007bff',
      'gelber-sack': '#ffc107',
      'sondermuell': '#dc3545',
      'altkleider': '#fd7e14',
    },
  },
})
