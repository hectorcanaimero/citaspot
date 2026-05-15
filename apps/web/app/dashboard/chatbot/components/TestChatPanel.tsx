'use client';

interface Props {
  onMessageSent: () => void;
  onCollapse: () => void;
}

export function TestChatPanel({ onCollapse }: Props) {
  return <div>TestChatPanel placeholder <button onClick={onCollapse}>x</button></div>;
}
