import { BookOpen } from "lucide-react";

// Scaffold only. The dictionary features are migrated from v1 in Phase 1.
export default function DictionaryPage() {
	return (
		<div className="mx-auto max-w-5xl px-4 py-8">
			<div className="mb-6 flex items-center gap-2">
				<BookOpen size={22} className="text-indigo-600 dark:text-indigo-400" />
				<h1 className="text-xl font-bold text-gray-900 dark:text-gray-100">Dictionary</h1>
			</div>
		</div>
	);
}
