# Sound Asset Attributions

## notification.mp3

- **Source**: Synthesized locally via `ffmpeg` (sine wave at 880 Hz, 150 ms, fade-out 70 ms, volume 0.3).
- **License**: CC0 (public domain) — no third-party material.
- **Command** used to generate:

  ```bash
  ffmpeg -y -f lavfi -i "sine=frequency=880:duration=0.15" \
    -af "afade=t=out:st=0.08:d=0.07,volume=0.3" \
    -ac 1 -ar 44100 -b:a 96k notification.mp3
  ```

- **Intended use**: Subtle "ping" played on the dashboard when a new appointment
  arrives via the realtime SSE stream. Volume kept low (0.3) so it does not
  startle staff. Duration kept short (≤ 200 ms) to avoid overlap with rapid
  consecutive notifications.
