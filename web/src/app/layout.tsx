import type { Metadata } from "next";
import { IBM_Plex_Sans, Space_Grotesk } from "next/font/google";
import type { ReactNode } from "react";

import { Providers } from "@/components/providers";
import "@/app/globals.css";

const plex = IBM_Plex_Sans({
  subsets: ["latin"],
  variable: "--font-body",
  weight: ["400", "500", "600", "700"],
});

const grotesk = Space_Grotesk({
  subsets: ["latin"],
  variable: "--font-display",
  weight: ["500", "700"],
});

export const metadata: Metadata = {
  title: "SlakeZAPI Console",
  description: "Painel moderno para operar a API SaaS de WhatsApp.",
};

export default function RootLayout({ children }: { children: ReactNode }) {
  return (
    <html lang="pt-BR">
      <body className={`${plex.variable} ${grotesk.variable} min-h-screen bg-ink text-mist`}>
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
