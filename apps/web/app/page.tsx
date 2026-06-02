import type { Metadata } from 'next';
import LandingPage from '@/components/marketing/LandingPage';

const TITLE = 'CitaSpot — Tu agenda con IA vía WhatsApp';
const DESCRIPTION =
  'Automatiza citas, responde clientes 24/7 y reduce no-shows. El asistente IA para negocios de salud y belleza en LATAM.';

export const metadata: Metadata = {
  title: TITLE,
  description: DESCRIPTION,
  applicationName: 'CitaSpot',
  keywords: [
    'agenda WhatsApp',
    'citas con IA',
    'asistente virtual WhatsApp',
    'gestión de citas',
    'no-shows',
    'salud y belleza',
    'CitaSpot',
  ],
  openGraph: {
    type: 'website',
    title: TITLE,
    description: DESCRIPTION,
    siteName: 'CitaSpot',
    locale: 'es_LA',
    images: [{ url: '/logo.jpeg', width: 1200, height: 630, alt: 'CitaSpot' }],
  },
  twitter: {
    card: 'summary_large_image',
    title: TITLE,
    description: DESCRIPTION,
    images: ['/logo.jpeg'],
  },
};

export default function HomePage() {
  return <LandingPage />;
}
