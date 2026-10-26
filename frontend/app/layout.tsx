import type { Metadata, Viewport } from "next";
import { Geist } from "next/font/google";
import "./globals.css";
import { ThemeProvider } from "@/contexts/ThemeContext";

const geist = Geist({
	variable: "--font-geist-sans",
	subsets: ["latin"],
});

export const metadata: Metadata = {
	title: "LyricfyJP — Learn Japanese with Music",
	description:
		"Learn Japanese using real song lyrics — furigana annotations, live karaoke sync, personal dictionary, and flashcards.",
};

export const viewport: Viewport = {
	width: "device-width",
	initialScale: 1,
};

// suppressHydrationWarning is needed because of the script below
// 1. Server sends HTML:           <html class="geist">            ← no "dark"
// 2. Browser reads <head>,
// our script runs:             <html class="geist dark">       ← script added "dark"
// 3. Page is drawn                (dark colors — no flash ✅)
// 4. React loads and hydrates.
// React's copy says:           <html class="geist">
// Screen says:                 <html class="geist dark"> → "Spot the difference!" ⚠️ warning
export default function RootLayout({ children }: LayoutProps<"/">) {
	return (
		<html lang="en" className={geist.variable} suppressHydrationWarning>
			<head>
				{/* Apply the saved theme before first paint to avoid a light-mode flash */}
				<script
					dangerouslySetInnerHTML={
                        {
						__html: `(function(){var t=localStorage.getItem('theme');if(t==='dark'||(!t&&window.matchMedia('(prefers-color-scheme: dark)').matches)){document.documentElement.classList.add('dark')}})()`,
					}}
				/>
			</head>
			<body className="min-h-screen bg-gray-50 font-sans antialiased dark:bg-gray-950">
				<ThemeProvider>{children}</ThemeProvider>
			</body>
		</html>
	);
}
