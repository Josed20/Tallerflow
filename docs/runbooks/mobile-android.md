# Android: build and pilot

The Vue application is packaged with Capacitor 8. The API is always the same HTTPS API used by the web application. A release APK cannot connect until José's mobile bearer session and CORS contract are merged.

## Local build

1. Install Node 24, Android Studio with Android SDK 36, and JDK 21.
2. Set `VITE_API_BASE_URL` to the public HTTPS origin, for example `https://tallerflow.pe`. Do not include `/api`.
3. Run `npm --prefix frontend ci` and `npm --prefix frontend run android:sync`.
4. Run `npm --prefix frontend run android:open`; select a connected Android device or emulator in Android Studio.

The Android app uses encrypted storage backed by Android Keystore for the opaque session token. The browser continues to use its cookie. Camera images are resized to at most 1600 px and encoded as JPEG before upload. Barcode scanning requires Google Play Services; while its module installs or access is unavailable, the operator can enter the printed order code manually.

## Signed release

The `Android APK` workflow runs on a `v*` tag. Set GitHub Actions secrets `TF_PUBLIC_BASE_URL`, `TF_ANDROID_KEYSTORE_B64`, `TF_ANDROID_KEYSTORE_PASSWORD`, `TF_ANDROID_KEY_ALIAS`, and `TF_ANDROID_KEY_PASSWORD`. The keystore must also be stored securely outside GitHub by José. Download the signed APK from the workflow artifact and test it on an older and a current Android phone before distributing it.

Never commit a `.jks`, `.keystore`, APK, password, or `.env`. The Android project ignores them. The app icons and splash can be generated after Lucero's resource files are merged; this branch intentionally retains Capacitor's provisional artwork.
