/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: ['./src/**/*.{html,ts}'],
  theme: {
    extend: {
      colors: {
        slate: {
          50: '#f8fafd',  // Gemini / Antigravity subtle surface
          100: '#f0f4f9', // Gemini sidebar & surface container
          200: '#dde3ea', // Gemini hairline border / outline-variant
          300: '#c4c7c5', // Material 3 outline
          400: '#747775', // Material 3 muted icon / placeholder
          500: '#5f6368', // Google secondary text
          600: '#444746', // Material 3 on-surface-variant
          700: '#2d3134', // Google medium-dark ink
          800: '#1f1f1f', // Gemini primary on-surface ink
          900: '#131314', // Gemini deep heading ink
          950: '#0b0c0e',
        },
        sky: {
          50: '#edf2fc',  // Gemini soft blue tint
          100: '#d3e3fd', // Gemini active pill container
          200: '#a8c7fa', // Gemini blue border
          300: '#7cacf8',
          400: '#4c8df6',
          500: '#1a73e8', // Google Blue
          600: '#0b57d0', // Gemini Primary Blue CTA
          700: '#0842a0', // Gemini Primary Blue hover
          800: '#062e6f',
          900: '#041e49', // Gemini on-secondary-container navy
          950: '#02122c',
        },
        canvas: {
          app: '#ffffff',
          subtle: '#f8fafd', // Gemini surface-low
        },
        surface: {
          card: '#ffffff',
          hover: '#f0f4f9', // Gemini surface-container
        },
        border: {
          subtle: '#dde3ea',
          strong: '#c4c7c5',
        },
        brand: {
          DEFAULT: '#0b57d0', // Gemini primary blue
          hover: '#0842a0',
        },
        status: {
          backlog: { bg: '#f0f4f9', text: '#444746', border: '#dde3ea' },
          todo: { bg: '#f0f4f9', text: '#1f1f1f', border: '#c4c7c5' },
          progress: { bg: '#fef7e0', text: '#b06000', border: '#fde293' },
          review: { bg: '#edf2fc', text: '#0b57d0', border: '#a8c7fa' },
          completed: { bg: '#e6f4ea', text: '#137333', border: '#ceead6' },
          cancelled: { bg: '#fce8e6', text: '#c5221f', border: '#fad2cf' },
        },
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'monospace'],
      },
      boxShadow: {
        card: '0 1px 2px 0 rgba(0, 0, 0, 0.03)',
      },
    },
  },
  plugins: [],
};
