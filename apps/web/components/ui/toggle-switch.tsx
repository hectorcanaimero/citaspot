'use client';

interface ToggleSwitchProps {
  checked: boolean;
  onCheckedChange: (checked: boolean) => void;
  activeColor?: string;
  size?: 'default' | 'sm';
}

const sizeClasses = {
  default: {
    track: 'h-5 w-9',
    knob: 'h-4 w-4',
  },
  sm: {
    track: 'h-4 w-8',
    knob: 'h-3 w-3',
  },
};

export function ToggleSwitch({
  checked,
  onCheckedChange,
  activeColor = 'bg-primary-600',
  size = 'default',
}: ToggleSwitchProps) {
  const s = sizeClasses[size];
  return (
    <button
      type="button"
      role="switch"
      aria-checked={checked}
      onClick={() => onCheckedChange(!checked)}
      className={`relative ${s.track} flex-shrink-0 overflow-hidden rounded-full transition-colors duration-200 focus:outline-none focus:ring-2 focus:ring-primary-400 focus:ring-offset-1 ${
        checked ? activeColor : 'bg-neutral-200'
      }`}
    >
      <span
        className={`absolute left-0.5 top-0.5 ${s.knob} rounded-full bg-white shadow transition-transform duration-200 ${
          checked ? 'translate-x-4' : 'translate-x-0'
        }`}
      />
    </button>
  );
}
