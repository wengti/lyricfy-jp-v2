import { Suspense } from "react";
import Navbar from "@/components/layout/Navbar";
import { requireSession } from "@/lib/auth/require-session";

// Layout for every page that requires login. The navbar is static and prerenders;
// the session check reads cookies, so it streams in behind a Suspense boundary.
export default function AppLayout({ children }: { children: React.ReactNode }) {
	return (
		<>
			<Navbar />
			<main>
				<Suspense>
					<AuthGate>{children}</AuthGate>
				</Suspense>
			</main>
		</>
	);
}

// Verifies the session with the backend before rendering the page
async function AuthGate({ children }: { children: React.ReactNode }) {
	await requireSession();
	return children;
}
