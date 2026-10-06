// Capture is complete only when every key pressed during this capture is released.
export function createShortcutCapture({save,cancel,error=()=>{},changed=()=>{}}) {
 let active=false;const down=new Set(),keys=new Set();
 const label=event=>{
  const code=event.code||event.key;
  if(/^Control/.test(code)||code==='Ctrl')return 'Ctrl';
  if(/^Shift/.test(code))return 'Shift';
  if(/^Alt/.test(code))return 'Alt';
  // Codes track physical release; letters must match the layout-aware native VK/keysym.
  if(/^[a-z0-9]$/i.test(event.key))return event.key.toUpperCase();
  if(/^Key[A-Z]$/.test(code))return code.slice(3);
  if(/^Digit[0-9]$/.test(code))return code.slice(5);
  if(/^Arrow/.test(code))return code.slice(5);
  if(code===' '||code==='Space')return 'Space';
  if(/^([A-Za-z0-9]|F([1-9]|1[0-2])|Tab|Enter|Backspace|Delete|Insert|Home|End|PageUp|PageDown|Left|Right|Up|Down)$/.test(code))return code.toUpperCase();
  return null;
 };
 const reset=()=>{active=false;down.clear();keys.clear();};
 const abort=()=>{if(!active)return;reset();cancel();};
 return {
  start(){reset();active=true;},cancel:abort,
  keydown(event){
   if(!active)return;event.preventDefault();event.stopImmediatePropagation?.();
   if(event.key==='Escape'){abort();return;}
   const key=label(event);if(!key){abort();error('This key is not supported. Use letters, numbers, F1–F12, Ctrl, Alt, Shift, Space, or navigation keys.');return;}
   down.add(event.code||event.key);keys.add(key);changed([...keys].join('+'));
  },
  keyup(event){
   if(!active)return;event.preventDefault();event.stopImmediatePropagation?.();
   down.delete(event.code||event.key);
   if(!down.size&&keys.size){const shortcut=[...keys].join('+');reset();save(shortcut);}
  }
 };
}
