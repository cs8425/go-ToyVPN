#include <stdio.h>
#include <stdlib.h>
#include <jni.h>
#include <android/log.h>

jstring jniNewStringUTF(JNIEnv *env, char* str);
const char* jniGetStringUTFChars(JNIEnv *env, jstring javaString);
void jniReleaseStringUTFChars(JNIEnv *env, jstring javaString, const char* nativeString);
void jniLog(JNIEnv *env, const char* tag, const char* nativeString);
void jniLog2(JNIEnv *env, jstring tagstr, jstring infostr);

