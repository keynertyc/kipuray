import { useEffect, useRef, useState } from "react";
import { Replayer } from "rrweb";
import type { eventWithTime } from "@rrweb/types";
import { Pause, Play, RotateCcw } from "lucide-react";
import { api } from "./api";
import type { BugCase } from "./types";
import "rrweb/dist/style.css";

export default function Replay() {
  const root = useRef<HTMLDivElement>(null),
    player = useRef<Replayer | null>(null),
    startAt = useRef(0);
  const [error, setError] = useState(""),
    [duration, setDuration] = useState(0),
    [position, setPosition] = useState(0),
    [playing, setPlaying] = useState(false),
    [width, setWidth] = useState(1280);
  useEffect(() => {
    let active = true;
    let observer: ResizeObserver | undefined;
    async function load() {
      try {
        const parts = location.pathname.split("/");
        const c = await api<BugCase>(
          parts[2] === "shared" ? "/shared/" + parts[3] : "/cases/" + parts[2],
        );
        if (!active || !root.current) return;
        if (c.events.length < 2) {
          setError(
            "This case does not contain a visual recording. Its scenario may still be executable.",
          );
          return;
        }
        const events = c.events as eventWithTime[];
        const replay = new Replayer(events, {
          root: root.current,
          showWarning: false,
          showDebug: false,
          mouseTail: false,
          UNSAFE_replayCanvas: false,
        });
        player.current = replay;
        setDuration(replay.getMetaData().totalTime);
        setWidth(c.width);
        replay.pause(0);
        observer = new ResizeObserver(() => {
          const wrapper =
            root.current?.querySelector<HTMLElement>(".replayer-wrapper");
          if (wrapper && root.current) {
            const scale = root.current.clientWidth / c.width;
            wrapper.style.transform = `scale(${scale})`;
            wrapper.style.transformOrigin = "top left";
            root.current.style.height = `${c.height * scale}px`;
          }
        });
        observer.observe(root.current);
        window.dispatchEvent(new Event("resize"));
      } catch (e) {
        if (active) setError(String(e));
      }
    }
    void load();
    return () => {
      active = false;
      observer?.disconnect();
      player.current?.destroy();
      player.current = null;
    };
  }, []);
  useEffect(() => {
    if (!playing) return;
    const timer = setInterval(() => {
      const value = Math.min(duration, performance.now() - startAt.current);
      setPosition(value);
      if (value >= duration) {
        setPlaying(false);
        player.current?.pause(duration);
      }
    }, 60);
    return () => clearInterval(timer);
  }, [playing, duration]);
  function toggle() {
    if (!player.current) return;
    if (playing) player.current.pause(position);
    else {
      const pos = position >= duration ? 0 : position;
      player.current.play(pos);
      startAt.current = performance.now() - pos;
    }
    setPlaying(!playing);
  }
  return (
    <div className="replay-page">
      {error ? (
        <div className="replay-empty">{error}</div>
      ) : (
        <div className="replay-viewport">
          <div ref={root} className="replay-root" style={{ maxWidth: width }} />
        </div>
      )}
      <div className="playback-controls">
        <button
          aria-label={playing ? "Pause recording" : "Play recording"}
          onClick={toggle}
          disabled={!!error}
        >
          {playing ? <Pause size={16} /> : <Play size={16} />}
        </button>
        <button
          aria-label="Restart recording"
          onClick={() => {
            player.current?.pause(0);
            setPlaying(false);
            setPosition(0);
          }}
          disabled={!!error}
        >
          <RotateCcw size={15} />
        </button>
        <input
          aria-label="Recording position"
          type="range"
          min="0"
          max={duration || 1}
          value={position}
          onChange={(e) => {
            const value = Number(e.target.value);
            setPosition(value);
            setPlaying(false);
            player.current?.pause(value);
          }}
        />
        <span>
          {(position / 1000).toFixed(1)} / {(duration / 1000).toFixed(1)} s
        </span>
      </div>
    </div>
  );
}
