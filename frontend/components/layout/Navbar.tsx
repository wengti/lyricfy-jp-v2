"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { BookOpen, LogOut, Moon, Sun } from "lucide-react";
import { useTheme } from "@/contexts/ThemeContext";
import { logout } from "@/app/actions/auth";
import { Wordmark } from "./Logo";

// More links (Lyrics, Browse, Flashcards) come back as their features are migrated
const NAV_LINKS = [{ href: "/dictionary", label: "Dictionary", icon: BookOpen }];

const ICON_BUTTON_CLASS =
	"rounded-md px-1.5 py-2 text-gray-500 transition-colors hover:bg-gray-100 sm:p-2 dark:text-gray-400 dark:hover:bg-gray-800";

export default function Navbar() {
	const pathname = usePathname();
	const { isDark, toggleTheme } = useTheme();

	return (
		<nav className="sticky top-0 z-50 border-b border-gray-200 bg-white dark:border-gray-800 dark:bg-gray-900">
			<div className="mx-auto flex max-w-5xl items-center justify-between px-4 py-3">
				<div className="flex flex-1 items-center">
					<Link href="/dictionary">
						<Wordmark className="text-lg" />
					</Link>
				</div>

				<div className="flex items-center gap-0 sm:gap-1">
					{NAV_LINKS.map(({ href, label, icon: Icon }) => {
						const active = pathname.startsWith(href);
						return (
							<Link
								key={href}
								href={href}
								title={label}
								className={`flex items-center gap-1.5 rounded-md px-1.5 py-2 text-sm font-medium transition-colors sm:px-3 ${
									active
										? "bg-indigo-50 text-indigo-700 dark:bg-indigo-900/30 dark:text-indigo-300"
										: "text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-400 dark:hover:bg-gray-800 dark:hover:text-gray-100"
								}`}
							>
								<Icon size={16} />
								<span className="hidden sm:inline">{label}</span>
							</Link>
						);
					})}
				</div>

				<div className="flex flex-1 items-center justify-end gap-0 sm:gap-1">
					<button
						onClick={toggleTheme}
						aria-label={isDark ? "Switch to light mode" : "Switch to dark mode"}
						className={`${ICON_BUTTON_CLASS} hover:text-gray-900 dark:hover:text-gray-100`}
					>
						{isDark ? <Sun size={18} /> : <Moon size={18} />}
					</button>
					<form action={logout}>
						<button
							type="submit"
							aria-label="Log out"
							title="Log out"
							className={`${ICON_BUTTON_CLASS} hover:text-red-600 dark:hover:text-red-400`}
						>
							<LogOut size={18} />
						</button>
					</form>
				</div>
			</div>
		</nav>
	);
}
