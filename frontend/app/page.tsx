import LoginForm from "@/components/auth/LoginForm";
import { LogoIcon, Wordmark } from "@/components/layout/Logo";

// Landing page: login. Logged-in visitors are redirected to /dictionary by proxy.ts.
export default function LoginPage() {
	return (
		<div className="flex min-h-screen flex-col items-center justify-center px-4">
			<div className="w-full max-w-sm">
				<div className="mb-8 flex flex-col items-center gap-2">
					<LogoIcon />
					<h1>
						<Wordmark className="text-2xl" />
					</h1>
					<p className="text-sm text-gray-500 dark:text-gray-400">Learn Japanese through music</p>
				</div>

				<div className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm sm:p-8 dark:border-gray-800 dark:bg-gray-900">
					<h2 className="mb-6 text-center text-lg font-semibold text-gray-900 dark:text-gray-100">
						Sign in to your account
					</h2>
					<LoginForm />
				</div>
			</div>
		</div>
	);
}
