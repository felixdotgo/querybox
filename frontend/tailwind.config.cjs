module.exports = {
  content: [
    "./index.html",
    // include Vue single-file components so Tailwind picks up classes used in templates
    "./src/**/*.{vue,js,ts,jsx,tsx,html}",
    "./public/**/*.css"
  ],
  theme: {
    extend: {},
  },
  plugins: [],
};
