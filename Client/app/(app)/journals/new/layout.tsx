export default function NewJournalLayout({
    children,
}: {
    children: React.ReactNode;
}) {
    return (
        <>
            {/* Darker background overlay for writing mode */}
            <div className="fixed inset-0 z-0 bg-black/30 pointer-events-none" />

            {/* Content */}
            <div className="relative z-10">
                {children}
            </div>
        </>
    );
}
