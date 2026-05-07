import type { Config } from "tailwindcss";

const config: Config = {
  content: ["./src/**/*.{js,ts,jsx,tsx,mdx}"],
  theme: {
    extend: {
      colors: {
        ink: "#08111f",
        mist: "#d6e4ff",
        line: "rgba(148, 163, 184, 0.18)",
        panel: "rgba(8, 17, 31, 0.78)",
        glow: "#4fd1ff",
        neon: "#7af7a7",
        ember: "#ffb86b",
        danger: "#ff6b7a"
      },
      boxShadow: {
        panel: "0 24px 80px rgba(0, 0, 0, 0.35)",
      },
      backgroundImage: {
        grid: "linear-gradient(rgba(148,163,184,0.08) 1px, transparent 1px), linear-gradient(90deg, rgba(148,163,184,0.08) 1px, transparent 1px)",
      }
    },
  },
  plugins: [],
};

export default config;
