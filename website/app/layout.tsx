import type { Metadata } from 'next';
import '@/styles/globals.css';

export const metadata: Metadata = {
  title: 'prr — AI-Powered PR Code Review CLI',
  description: 'Run AI-powered code reviews on GitHub pull requests or local git branches. Human-like comments from Claude and GPT, structured markdown output, one command.',
  openGraph: {
    title: 'prr — AI-Powered PR Code Review CLI',
    description: 'Run AI-powered code reviews on GitHub pull requests or local git branches. Human-like comments from Claude and GPT, structured markdown output, one command.',
    url: 'https://prr.dotbrains.io',
    siteName: 'prr',
    images: [
      {
        url: '/og-image.svg',
        width: 1200,
        height: 630,
        alt: 'prr — AI-Powered PR Code Review CLI',
      },
    ],
    locale: 'en_US',
    type: 'website',
  },
  twitter: {
    card: 'summary_large_image',
    title: 'prr — AI-Powered PR Code Review CLI',
    description: 'Run AI-powered code reviews on GitHub pull requests or local git branches. Human-like comments from Claude and GPT, structured markdown output, one command.',
    images: ['/og-image.svg'],
  },
  icons: {
    icon: [
      {
        url: '/favicon.svg',
        type: 'image/svg+xml',
      },
    ],
    apple: [
      {
        url: '/favicon.svg',
        type: 'image/svg+xml',
      },
    ],
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <meta charSet="UTF-8" />
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
      </head>
      <body>{children}</body>
    </html>
  );
}
