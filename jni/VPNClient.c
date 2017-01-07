#include "VPNClient.h"

jstring jniNewStringUTF(JNIEnv *env, char* str) {
    return (*env)->NewStringUTF(env, str);
}

const char* jniGetStringUTFChars(JNIEnv *env, jstring javaString) {
	return (*env)->GetStringUTFChars(env, javaString, 0);
}

void jniReleaseStringUTFChars(JNIEnv *env, jstring javaString, const char* nativeString) {
	(*env)->ReleaseStringUTFChars(env, javaString, nativeString);
}

void jniLog(JNIEnv *env, const char* tag, const char* nativeString) {
	jclass class = (*env)->FindClass(env, "android/util/Log");
//	jmethodID method = (*env)->GetMethodID(env,class, "i", "(Ljava/lang/String;Ljava/lang/String;)I");
	jmethodID method = (*env)->GetStaticMethodID(env,class, "i", "(Ljava/lang/String;Ljava/lang/String;)I");

	jstring tagstr = (*env)->NewStringUTF(env, tag);
	jstring infostr = (*env)->NewStringUTF(env, nativeString);

	jint ret = (jint)(*env)->CallStaticIntMethod(env, class, method, tagstr, infostr);

	(*env)->ReleaseStringUTFChars(env, tagstr, tag);
	(*env)->ReleaseStringUTFChars(env, infostr, nativeString);
}

void jniLog2(JNIEnv *env, jstring tagstr, jstring infostr) {
	jclass class = (*env)->FindClass(env, "android/util/Log");
//	jmethodID method = (*env)->GetMethodID(env,class, "i", "(Ljava/lang/String;Ljava/lang/String;)I");
	jmethodID method = (*env)->GetStaticMethodID(env,class, "i", "(Ljava/lang/String;Ljava/lang/String;)I");
	jint ret = (jint)(*env)->CallStaticIntMethod(env, class, method, tagstr, infostr);
}

