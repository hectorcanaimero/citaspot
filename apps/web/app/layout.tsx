import type { Metadata } from 'next';
import './globals.css';
import { LanguageProvider } from '@/lib/i18n';

export const metadata: Metadata = {
  title: 'CitaSpot — Gestión de citas con IA',
  description: 'Automatiza tu agenda y atiende clientes por WhatsApp con inteligencia artificial.',
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  // lang="es" es el default SSR; LanguageProvider lo actualiza en el cliente
  // según localStorage o el idioma del navegador.
  return (
    <html lang="es">
      <body>
        <LanguageProvider>{children}</LanguageProvider>
      </body>
    </html>
  );
}
