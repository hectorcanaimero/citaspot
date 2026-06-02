import { Nav } from './Nav';
import { Hero } from './Hero';
import { Marquee } from './Marquee';
import { ForWho } from './ForWho';
import { PainPoints } from './PainPoints';
import { HowItWorks } from './HowItWorks';
import { DashboardPreview } from './DashboardPreview';
import { WaitlistCommunity } from './WaitlistCommunity';
import { FinalCTA } from './FinalCTA';
import { Footer } from './Footer';
import { StickyMobileCTA } from './StickyMobileCTA';

export default function LandingPage() {
  return (
    <div className="bg-white text-lp-ink">
      <Nav />
      <main>
        <Hero />
        <Marquee />
        <ForWho />
        <PainPoints />
        <HowItWorks />
        <DashboardPreview />
        <WaitlistCommunity />
        <FinalCTA />
      </main>
      <Footer />
      <StickyMobileCTA />
    </div>
  );
}
