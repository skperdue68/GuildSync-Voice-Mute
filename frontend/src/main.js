import {io} from 'socket.io-client';
import {createVoiceSession} from './voice-session.js';
import {createAuthCoordinator} from './auth-state.js';
import './style.css';
const api=()=>window.go.main.App;
const auth=createAuthCoordinator();
let session=null,socket=null,settings={enabled:false,shortcut:'Ctrl+M'},capturing=false,busy=false;
document.querySelector('#app').innerHTML=`<h1>GuildSync Voice Mute</h1><p class="intro">Hold your shortcut to mute lower-ranked members in your Discord voice channel.</p><section><p id="account">Not signed in</p><div class="buttons"><button id="login">Sign in with Discord</button><button id="logout" hidden>Sign out</button></div></section><section><label class="toggle"><input type="checkbox" id="enabled"> Enable global shortcut</label><p>Shortcut: <strong id="shortcut">Ctrl+M</strong></p><div class="buttons"><button id="capture">Set shortcut</button><button id="default" class="secondary">Return to Default</button></div><p class="hint" id="captureHelp">Ctrl, Alt or Shift + letter, number or F1–F12.</p></section><p id="status" role="status" aria-live="polite">Starting…</p><p class="hint">Moderator mutes are preserved. Equal and higher ranks are protected. Release the shortcut to end your mute request.</p>`;
const el=id=>document.getElementById(id);
function status(message){el('status').textContent=String(message||'');}
function render(){el('account').textContent=session?.logged_in?`Signed in as ${session.user.display_name||session.user.username} (${session.user.role})`:'Not signed in';el('login').hidden=Boolean(session?.logged_in);el('logout').hidden=!session?.logged_in;el('shortcut').textContent=settings.shortcut;el('enabled').checked=settings.enabled;el('enabled').disabled=busy||!session?.allowed||!['user','admin'].includes(session?.user?.role);el('capture').disabled=busy;el('default').disabled=busy;}
const controller=createVoiceSession({getSocket:()=>socket,notify:status});
async function disableListener(){controller.stop();await api().SetShortcutActive(false);}
async function activate(){if(!socket?.connected||!settings.enabled||capturing)return;try{await api().SetShortcutActive(true);status('Ready. Hold your shortcut while in a Discord voice channel.');}catch(error){status(error);}}
function clearSocket(){controller.stop();socket?.disconnect();socket=null;}
async function connect(s){const epoch=auth.invalidate();await disableListener();if(epoch!==auth.current())return;clearSocket();session=s;render();if(!s?.allowed||!s.token){status('Sign in with an approved GuildSync User or Admin account.');return;}socket=io(s.socket_url,{auth:{token:s.token}});socket.on('connect',activate);socket.on('disconnect',()=>{controller.onDisconnect();api().SetShortcutActive(false).catch(status);status('Disconnected. Reconnecting…');});socket.on('connect_error',()=>{controller.onDisconnect();api().SetShortcutActive(false).catch(status);status('Connection unavailable. Check that your account is approved and the server is online.');});}
async function save(next){busy=true;render();try{await disableListener();await api().SetShortcutSettings(next);settings=await api().GetShortcutSettings();await activate();}catch(error){status(error);}finally{busy=false;render();}}
el('login').onclick=async()=>{auth.invalidate();try{await api().StartDiscordLogin();status('Complete Discord login in your browser.');}catch(error){status(error);}};
el('logout').onclick=async()=>{auth.invalidate();try{await disableListener();clearSocket();await api().Logout();session=null;render();status('Signed out.');}catch(error){status(error);}};
el('enabled').onchange=()=>save({...settings,enabled:el('enabled').checked});
el('default').onclick=()=>save({enabled:false,shortcut:'Ctrl+M'});
el('capture').onclick=async()=>{try{await disableListener();capturing=true;el('captureHelp').textContent='Press a combination now. Escape cancels.';el('capture').textContent='Listening…';}catch(error){status(error);}};
function endCapture(){capturing=false;el('capture').textContent='Set shortcut';el('captureHelp').textContent='Ctrl, Alt or Shift + letter, number or F1–F12.';}
window.addEventListener('keydown',event=>{if(!capturing)return;event.preventDefault();if(event.key==='Escape'){endCapture();activate();return;}if(['Control','Alt','Shift','Meta'].includes(event.key))return;const mods=[];if(event.ctrlKey)mods.push('Ctrl');if(event.altKey)mods.push('Alt');if(event.shiftKey)mods.push('Shift');if(event.metaKey||!mods.length){status('Use Ctrl, Alt or Shift plus another key.');return;}const key=event.key.toUpperCase();if(!/^([A-Z0-9]|F([1-9]|1[0-2]))$/.test(key)){status('Use a letter, number or F1–F12.');return;}endCapture();save({...settings,shortcut:[...mods,key].join('+')});});
window.runtime.EventsOn('voice-shortcut-edge',pressed=>controller.onEdge(pressed));
window.runtime.EventsOn('login-complete',s=>connect(s).catch(status));
window.runtime.EventsOn('login-error',status);
window.runtime.EventsOn('shortcut-error',message=>{controller.stop();api().SetShortcutActive(false).catch(()=>{});status(message);});
window.addEventListener('beforeunload',()=>{controller.stop();socket?.disconnect();});
// Revalidate saved accounts while running. Socket-side checks remain authoritative for every request.
let verifying=false;
setInterval(async()=>{if(!session?.logged_in||verifying)return;verifying=true;try{await auth.run(()=>api().GetSession(),fresh=>{if(!fresh.logged_in)throw Error('Sign in again.');session=fresh;render();},async error=>{const epoch=auth.invalidate();await disableListener();if(epoch!==auth.current())return;clearSocket();session=null;render();status(error);});}finally{verifying=false;}},30000);
async function initialize(){try{settings=await api().GetShortcutSettings();await connect(await api().GetSession());}catch(error){render();status(error);}}
initialize();
