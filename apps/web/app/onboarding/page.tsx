import { redirect } from 'next/navigation';

// Redirigir al primer paso del onboarding.
export default function OnboardingIndex() {
  redirect('/onboarding/services');
}
