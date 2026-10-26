"use client";

import { useActionState } from "react";
import { login, type LoginState } from "@/app/actions/auth";

const INPUT_CLASS =
	"w-full rounded-lg border border-gray-300 px-3 py-2 text-base outline-none focus:border-indigo-500 focus:ring-2 focus:ring-indigo-200 dark:border-gray-600 dark:bg-gray-800 dark:text-gray-100 dark:placeholder-gray-500 dark:focus:ring-indigo-800";

const initialState: LoginState = { error: null };

export default function LoginForm() {
	const [state, formAction, pending] = useActionState(login, initialState);

	return (
		<>
			{state.error && (
				<div
					role="alert"
					className="mb-4 rounded-lg bg-red-50 px-4 py-3 text-sm text-red-700 dark:bg-red-900/30 dark:text-red-300"
				>
					{state.error}
				</div>
			)}

			<form action={formAction} className="space-y-4">
				<div>
					<label htmlFor="email" className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
						Email
					</label>
					<input
						id="email"
						name="email"
						type="email"
						autoComplete="email"
						required
						className={INPUT_CLASS}
						placeholder="you@example.com"
					/>
				</div>
				<div>
					<label htmlFor="password" className="mb-1.5 block text-sm font-medium text-gray-700 dark:text-gray-300">
						Password
					</label>
					<input
						id="password"
						name="password"
						type="password"
						autoComplete="current-password"
						required
						className={INPUT_CLASS}
						placeholder="••••••••"
					/>
				</div>
				<button
					type="submit"
					disabled={pending}
					className="w-full rounded-lg bg-indigo-600 py-2 text-sm font-medium text-white transition-colors hover:bg-indigo-700 disabled:opacity-60"
				>
					{pending ? "Signing in…" : "Sign in"}
				</button>
			</form>
		</>
	);
}
