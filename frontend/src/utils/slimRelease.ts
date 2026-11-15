export interface SlimRelease { tag_name?: string; body?: string; draft?: boolean }

export function latestSlimRelease(releases: readonly SlimRelease[]): SlimRelease | undefined {
  let best: { release: SlimRelease; version: number[] } | undefined
  for (const release of releases) {
    if (release.draft) continue
    const match = /^v?(\d+)\.(\d+)\.(\d+)-slim$/.exec(release.tag_name || '')
    if (!match) continue
    const version = match.slice(1).map(Number)
    const newer = !best || version.some((part, index) => part > best!.version[index] && version.slice(0, index).every((previous, position) => previous === best!.version[position]))
    if (newer) best = { release, version }
  }
  return best?.release
}
