import {io} from 'socket.io-client';
import {createVoiceSession} from './voice-session.js';
import {createShortcutCapture} from './shortcut-capture.js';
import {createAuthCoordinator,canEnableVoiceSession} from './auth-state.js';
import './style.css';
import {createShortcutActivation} from './shortcut-activation.js';
const activation=createShortcutActivation();
let availability={available:true,reason:''};
const api=()=>window.go.main.App;
const auth=createAuthCoordinator();
let session=null,socket=null,settings={enabled:false,shortcut:'Ctrl+M'},capturing=false,busy=false;
document.querySelector('#app').innerHTML=`<h1>GuildSync Voice Mute</h1><p id="appVersion" class="hint"></p><p class="intro">Hold your shortcut to mute lower-ranked members in your Discord voice channel.</p><section><p id="account">Not signed in</p><div class="buttons"><button id="login">Sign in with Discord</button><button id="logout" hidden>Sign out</button></div></section><section><label class="toggle"><input type="checkbox" id="enabled"> Enable global shortcut</label><p>Shortcut: <strong id="shortcut">Ctrl+M</strong></p><div class="buttons"><button id="capture">Set shortcut</button><button id="default" class="secondary">Return to Default</button></div><p id="macPermissions" class="hint" hidden>On Mac, enabling mute requests Input Monitoring. Allow GuildSync Voice Mute in System Settings → Privacy &amp; Security → Input Monitoring, then quit and reopen this app. On macOS 12, use System Preferences → Security &amp; Privacy → Privacy → Input Monitoring. This permission is optional; leave mute disabled if you do not want to grant it.</p><p class="hint" id="captureHelp">Press one or more keys, then release all keys to save. Escape cancels.</p></section><p id="availability" role="status"></p><p id="status" role="status" aria-live="polite">Starting…</p><p class="hint">Moderator mutes are preserved. Equal and higher ranks are protected. Release the shortcut to end your mute request.</p>`;
const el=id=>document.getElementById(id);
function status(message){el('status').textContent=String(message||'');}
function render(){el('availability').textContent=availability.reason;el('account').textContent=session?.logged_in?`Signed in as ${session.user.display_name||session.user.username}`:'Not signed in';el('login').hidden=Boolean(session?.logged_in);el('logout').hidden=!session?.logged_in;el('shortcut').textContent=settings.shortcut;el('enabled').checked=settings.enabled;el('enabled').disabled=busy||!availability.available||!canEnableVoiceSession(session);el('capture').disabled=busy;el('default').disabled=busy;}
const controller=createVoiceSession({getSocket:()=>socket,notify:status});
async function disableListener(){activation.invalidate();controller.stop();await api().SetShortcutActive(false);}
function canActivate(){return Boolean(socket?.connected&&settings.enabled&&!capturing&&availability.available);}
async function activate(){if(!canActivate())return;try{if(await activation.activate(()=>api().SetShortcutActive(true),canActivate))status('Ready. Hold your shortcut while in a Discord voice channel.');}catch(error){availability=await api().GetShortcutAvailability();render();status(availability.reason||error);}}
function clearSocket(){controller.stop();socket?.disconnect();socket=null;}
async function connect(s){const epoch=auth.invalidate();await disableListener();if(epoch!==auth.current())return;clearSocket();session=s;render();if(!s?.allowed||!s.token){status('Sign in with Discord. Your server role determines mute access.');return;}socket=io(s.socket_url,{auth:{source:'voice-mute',token:s.token}});socket.on('connect',()=>{activation.invalidate();void activate();});socket.on('disconnect',()=>{activation.invalidate();controller.onDisconnect();api().SetShortcutActive(false).catch(status);status('Disconnected. Reconnecting…');});socket.on('connect_error',()=>{activation.invalidate();controller.onDisconnect();api().SetShortcutActive(false).catch(status);status('Connection unavailable. Check that your Discord login is current and the server is online.');});}
async function save(next){activation.retry();busy=true;render();try{await disableListener();await api().SetShortcutSettings(next);settings=await api().GetShortcutSettings();await activate();}catch(error){status(error);}finally{busy=false;render();}}
el('login').onclick=async()=>{auth.invalidate();try{await api().StartDiscordLogin();status('Complete Discord login in your browser.');}catch(error){status(error);}};
el('logout').onclick=async()=>{auth.invalidate();try{await disableListener();clearSocket();await api().Logout();session=null;render();status('Signed out.');}catch(error){status(error);}};
el('enabled').onchange=()=>save({...settings,enabled:el('enabled').checked});
el('default').onclick=()=>save({enabled:false,shortcut:'Ctrl+M'});
el('capture').onclick=async()=>{try{await disableListener();capturing=true;shortcutCapture.start();el('captureHelp').textContent='Press one or more keys; release all keys to save. Escape cancels.';el('capture').textContent='Listening…';}catch(error){status(error);}};
function endCapture(){capturing=false;el('capture').textContent='Set shortcut';el('captureHelp').textContent='Press one or more keys, then release all keys to save. Escape cancels.';}
const shortcutCapture=createShortcutCapture({
 save:shortcut=>{endCapture();void save({...settings,shortcut});},
 cancel:()=>{endCapture();void activate();},error:status,
 changed:shortcut=>{el('captureHelp').textContent=`${shortcut} — release all keys to save`;}
});
window.addEventListener('keydown',event=>shortcutCapture.keydown(event));
window.addEventListener('keyup',event=>shortcutCapture.keyup(event));
window.addEventListener('blur',()=>shortcutCapture.cancel());
window.runtime.EventsOn('voice-shortcut-edge',pressed=>controller.onEdge(pressed));
window.runtime.EventsOn('login-complete',s=>connect(s).catch(status));
window.runtime.EventsOn('login-error',status);
window.runtime.EventsOn('shortcut-error',message=>{activation.fail();controller.stop();api().SetShortcutActive(false).catch(()=>{});status(message);});
window.addEventListener('beforeunload',()=>{controller.stop();socket?.disconnect();});
// Revalidate saved accounts while running. Socket-side checks remain authoritative for every request.
let verifying=false;
setInterval(async()=>{if(!session?.logged_in||verifying)return;verifying=true;try{await auth.run(()=>api().GetSession(),fresh=>{if(!fresh.logged_in)throw Error('Sign in again.');session=fresh;render();},async error=>{const epoch=auth.invalidate();await disableListener();if(epoch!==auth.current())return;clearSocket();session=null;render();status(error);});}finally{verifying=false;}},30000);
async function initialize(){try{el('appVersion').textContent='Version '+await api().GetAppVersion();el('macPermissions').hidden=await api().GetPlatform()!=='darwin';settings=await api().GetShortcutSettings();await connect(await api().GetSession());}catch(error){render();status(error);}}
initialize();
