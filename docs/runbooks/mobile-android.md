# Android: build and pilot

The Vue application is packaged with Capacitor 8. The release APK connects to `https://tallerflowpe.com` through the mobile bearer session and CORS contract already merged in Sprint 3.

## Registro de talleres desde Android

El registro y el reenvío del correo de activación aceptan tanto el origen web
(`TF_ALLOWED_ORIGIN`) como los orígenes exactos de la app (`TF_MOBILE_ORIGINS`).
En producción, Android utiliza `https://localhost`. La configuración se comparte
con el inicio de sesión móvil; no debe sustituirse por un comodín.

Estos dos endpoints no crean una sesión ni emiten cookies. Se mantienen la
validación de datos y los límites por IP. La activación se completa abriendo el
enlace HTTPS del correo en el navegador e introduciendo la contraseña registrada;
después se puede iniciar sesión en Android.

Si el registro devuelve `ORIGIN_INVALID` desde Android y funciona desde la web,
comprobar que el handler de registro recibe `TF_MOBILE_ORIGINS`. Esta corrección
requiere desplegar la API, sin recompilar ni publicar otro AAB.

## Vista previa local por USB

Esta modalidad sirve para recorrer las pantallas actuales, iniciar sesión en el Docker local y probar los botones nativos de cámara y escáner. Usa temporalmente la sesión web con cookie; no prueba el token bearer ni el servidor final. El celular y la laptop se conectan por USB con depuración habilitada. No hace falta conocer la IP de ninguno ni estar en la misma red Wi-Fi.

En PowerShell, desde la raíz del repositorio:

```powershell
docker compose --profile app up -d --build --wait
& "$env:LOCALAPPDATA\Android\Sdk\platform-tools\adb.exe" devices
& "$env:LOCALAPPDATA\Android\Sdk\platform-tools\adb.exe" reverse tcp:8080 tcp:8080
$env:TF_ANDROID_LOCAL_PREVIEW = "1"
Remove-Item Env:VITE_API_BASE_URL -ErrorAction SilentlyContinue
npm --prefix frontend run android:preview
npm --prefix frontend run android:open
```

En Android Studio selecciona el dispositivo y pulsa **Run**. La app abre `http://localhost:8080` del dispositivo, redirigido por USB a Docker en la laptop. Si desconectas el cable, repite `adb reverse`. Al cambiar código Vue, reconstruye el frontend de Docker; al cambiar código nativo, vuelve a sincronizar y ejecutar Android. La configuración HTTP solo se usa en la vista previa y el permiso de tráfico sin cifrar existe solo en el manifiesto Android de depuración.

Para compilar el APK de depuracion desde PowerShell sin abrir Android Studio, apunta `JAVA_HOME` a JDK 21 y `ANDROID_HOME` al SDK instalado:

```powershell
$env:JAVA_HOME = "C:\Program Files\Java\jdk-21"
$env:ANDROID_HOME = "$env:LOCALAPPDATA\Android\Sdk"
Push-Location frontend/android
.\gradlew.bat assembleDebug --no-daemon
Pop-Location
```

El archivo resultante es `frontend/android/app/build/outputs/apk/debug/app-debug.apk`. El paquete debug es `pe.tallerflow.app.debug`, se llama «TallerFlow local» y se instala junto a la versión de Play Store. Solo la vista previa creada con `TF_ANDROID_LOCAL_PREVIEW=1` puede usar Docker local por USB; el APK de produccion requiere la API HTTPS y la sesion movil de Jose.

Para volver al build móvil normal en la misma terminal, quita `TF_ANDROID_LOCAL_PREVIEW`, configura una URL HTTPS real y ejecuta `android:sync`:

```powershell
Remove-Item Env:TF_ANDROID_LOCAL_PREVIEW -ErrorAction SilentlyContinue
$env:VITE_API_BASE_URL = "https://tallerflowpe.com"
npm --prefix frontend run android:sync
```

## Local build

1. Install Node 24, Android Studio with Android SDK 36, and JDK 21.
2. Set `VITE_API_BASE_URL` to `https://tallerflowpe.com`. Do not include `/api`.
3. Run `npm --prefix frontend ci` and `npm --prefix frontend run android:sync`.
4. Run `npm --prefix frontend run android:open`; select a connected Android device or emulator in Android Studio.

The Android app uses encrypted storage backed by Android Keystore for the opaque session token. The browser continues to use its cookie. Camera images are resized to at most 1600 px and encoded as JPEG before upload. Barcode scanning requires Google Play Services; while its module installs or access is unavailable, the operator can enter the printed order code manually.

## Signed release

The `Android APK` workflow runs manually or on a `v*` tag and targets `https://tallerflowpe.com`. Set GitHub Actions secrets `TF_ANDROID_KEYSTORE_B64`, `TF_ANDROID_KEYSTORE_PASSWORD`, `TF_ANDROID_KEY_ALIAS`, and `TF_ANDROID_KEY_PASSWORD`. The keystore must also be stored securely outside GitHub by José; losing it prevents signing future updates with the same identity. The workflow builds both a signed APK and an Android App Bundle (AAB), verifies both signatures, and prints their SHA-256 digests. Download the APK artifact `TallerFlow-<run_number>-android` and test it on an older and a current Android phone before distributing it. For Google Play, download `TallerFlow-android-play-bundle`, containing `app-release.aab` from `frontend/android/app/build/outputs/bundle/release/`. A GitHub Actions artifact is for the team and is not a permanent public download URL for the landing page; choose an approved release host before Lucero links it.

Never commit a `.jks`, `.keystore`, APK, password, or `.env`. The Android project ignores them. Check the installed icon and splash on both test phones before release.

## Optimización y avisos de Play Console

La compilación release aplica R8 y reducción de recursos. CI compila tanto debug
como release; el workflow firmado conserva `mapping.txt` en el artefacto
`TallerFlow-android-mapping` para interpretar fallas de código ofuscado. El AAB
incluye los metadatos de desofuscación generados por Android Gradle Plugin.

La revisión directa de Console del 24 de noviembre de 2026 identificó en 1.0.1:

- Ofuscación del 1 %, con fecha indicada de febrero de 2027. Se habilitó R8;
  el porcentaje nuevo debe verificarse tras el análisis del AAB en Play.
- APIs antiguas en Material y `IONCAMRImageEditorActivity`. Material se actualizó
  a 1.14.0, que protege esas llamadas en Android 15+. El editor de Ionic todavía
  usa `getStatusBarColor` y `getNavigationBarColor` en el código publicado de
  1.0.3; no se considera eliminado este aviso upstream. La captura de TallerFlow
  no solicita edición de fotos. No se suprimen diagnósticos para ocultarlo.
- Orientación vertical heredada de `GmsBarcodeScanningDelegateActivity`. El
  manifiesto de la app reemplaza esa restricción por orientación no especificada
  y permite redimensionar la actividad. Debe comprobarse el escaneo en vertical,
  horizontal y pantalla grande antes de distribuir.

Referencias: [R8](https://developer.android.com/build/enable-app-optimization),
[corrección de Material](https://github.com/material-components/material-components-android/issues/4507)
y [fuentes publicadas de Ionic](https://repo1.maven.org/maven2/io/ionic/libs/ioncamera-android/1.0.3/).

## Inicio y marca móvil

Android abre el inicio de sesión al arrancar sin sesión, incluso si está disponible el onboarding inicial; con sesión vigente abre el panel correspondiente a su rol, salvo que deba cambiar la contraseña primero. La web conserva la landing. La ruta pública `/eliminar-cuenta` explica cómo solicitar el borrado por `tallerflow.pe@gmail.com` sin iniciar sesión y también es accesible con sesión iniciada; está enlazada desde la política de privacidad y la sección Cuenta y privacidad del menú de usuario. La solicitud se envía desde el correo registrado, con el asunto `Eliminar mi cuenta de TallerFlow`, el nombre del taller y el alcance (cuenta personal o taller completo, si es propietario). El equipo debe verificar la identidad y el alcance y confirmar la tramitación por correo: esta página no ejecuta una purga automática.

Los íconos adaptativos, tradicionales y el splash usan `frontend/public/logo-symbol-on-dark.svg`. Para regenerarlos: `python scripts/generate_android_brand.py` desde la raíz (requiere Python e Inkscape en PATH o la variable `INKSCAPE` con la ruta del ejecutable de Inkscape). El generador Go antiguo no se debe usar para los recursos Android. Después ejecutar `npm --prefix frontend run android:sync` con la configuración HTTPS del build móvil normal indicada arriba. La versión 1.1.0 usa versionCode 3. La aplicación instalada se actualiza mediante un AAB nuevo en Google Play; desplegar la web no reemplaza los recursos incluidos en las instalaciones Android anteriores.
