// Layout del onboarding — sin sidebar, centrado, minimalista.
export default function OnboardingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className="min-h-screen bg-neutral-50">
      {/* Header simple */}
      <header className="flex h-14 items-center border-b border-neutral-100 bg-white px-6">
        <div className="flex items-center gap-2">
          <div className="flex h-7 w-7 items-center justify-center rounded-lg bg-primary-600 text-white text-sm font-bold">
            A
          </div>
          <span className="text-sm font-semibold text-neutral-900">CitaSpot</span>
        </div>
      </header>
      <main>{children}</main>
    </div>
  );
}
