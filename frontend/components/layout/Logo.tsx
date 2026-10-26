import { Music } from "lucide-react";

export function Wordmark({ className = "" }: { className?: string }) {
	return (
		<span className={`font-bold tracking-tight text-gray-900 dark:text-gray-100 ${className}`}>
			Lyricfy<span className="text-indigo-600 dark:text-indigo-400">JP</span>
		</span>
	);
}

// App icon tile, matching app/icon.png
export function LogoIcon({ size = 48 }: { size?: number }) {
	return (
		<div
			className="flex items-center justify-center rounded-xl bg-indigo-600"
			style={{ width: size, height: size }}
		>
			<Music size={size / 2} className="text-white" />
		</div>
	);
}
