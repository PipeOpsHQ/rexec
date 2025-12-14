export type XtermCoreModules = {
  Terminal: typeof import("@xterm/xterm").Terminal;
  FitAddon: typeof import("@xterm/addon-fit").FitAddon;
  Unicode11Addon: typeof import("@xterm/addon-unicode11").Unicode11Addon;
  WebLinksAddon: typeof import("@xterm/addon-web-links").WebLinksAddon;
};

export type XtermWebglModule = {
  WebglAddon: typeof import("@xterm/addon-webgl").WebglAddon;
};

let xtermCssPromise: Promise<unknown> | null = null;
let xtermCorePromise: Promise<XtermCoreModules> | null = null;
let xtermWebglPromise: Promise<XtermWebglModule> | null = null;

export async function loadXtermCore(): Promise<XtermCoreModules> {
  if (!xtermCorePromise) {
    xtermCorePromise = (async () => {
      if (!xtermCssPromise) {
        xtermCssPromise = import("@xterm/xterm/css/xterm.css");
      }
      await xtermCssPromise;

      const [xterm, fit, unicode11, webLinks] = await Promise.all([
        import("@xterm/xterm"),
        import("@xterm/addon-fit"),
        import("@xterm/addon-unicode11"),
        import("@xterm/addon-web-links"),
      ]);

      return {
        Terminal: xterm.Terminal,
        FitAddon: fit.FitAddon,
        Unicode11Addon: unicode11.Unicode11Addon,
        WebLinksAddon: webLinks.WebLinksAddon,
      };
    })().catch((err) => {
      // Allow retry on transient failures
      xtermCorePromise = null;
      throw err;
    });
  }

  return xtermCorePromise;
}

export async function loadXtermWebgl(): Promise<XtermWebglModule> {
  if (!xtermWebglPromise) {
    xtermWebglPromise = import("@xterm/addon-webgl")
      .then((mod) => ({ WebglAddon: mod.WebglAddon }))
      .catch((err) => {
        xtermWebglPromise = null;
        throw err;
      });
  }

  return xtermWebglPromise;
}

