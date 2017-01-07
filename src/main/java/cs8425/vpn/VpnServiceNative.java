/*
 * Copyright (C) 2011 The Android Open Source Project
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *      http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cs8425.vpn;

import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.net.VpnService;
import android.os.Handler;
import android.os.Message;
import android.os.ParcelFileDescriptor;
import android.util.Log;
import android.widget.Toast;

import java.net.InetSocketAddress;
import java.nio.channels.SocketChannel;
import java.nio.ByteBuffer;

public class VpnServiceNative extends VpnService implements Handler.Callback, Runnable {
	private static final String TAG = "ToyVpnServiceNative";

	private String mServerAddress;
	private String mServerPort;
	private String mSharedSecret;
	private PendingIntent mConfigureIntent;

	private static Handler mHandler;
	private static Thread mThread;

	private ParcelFileDescriptor mInterface;
	private String mParameters;
	private SocketChannel tunnel;
	private ParcelFileDescriptor tunnelFd;

	private static boolean isRunning = false;
	private static boolean isConnected = false;

	// We use a timer to determine the status of the tunnel. It
	// works on both sides. A positive value means sending, and
	// any other means receiving. We start with receiving.
	int timer = 0; //synchronized

	// Assume that we did not make any progress in this iteration.
	boolean idle = true; //synchronized

	static {
		//System.load("VPNClient");
		System.loadLibrary("VPNClient");
	}

	public static native long Multiply(long x, long y);
	public static native void SetTunFD(int fd);
	public static native void SetSocketFD(int fd);
	public static native int Dump();
	public static native String Handshake(String parm);
	public static native void Loop();
	public static native void Stop();

	@Override
	public int onStartCommand(Intent intent, int flags, int startId) {
		Log.i(TAG, "Got onStartCommand() " + flags + ", " + startId);
		// The handler is only used to show messages.
		if (mHandler == null) {
			mHandler = new Handler(this);
		}

		// Stop the previous session by interrupting the thread.
		if (mThread != null) {
			mThread.interrupt();
		}

		// Extract information from the intent.
		String prefix = getPackageName();
		mServerAddress = intent.getStringExtra(prefix + ".ADDRESS");
		mServerPort = intent.getStringExtra(prefix + ".PORT");
		mSharedSecret = intent.getStringExtra(prefix + ".SECRET");

		// Start a new session by creating a new thread.
		mThread = new Thread(this, "ToyVpnThread");
		mThread.start();

		isRunning = true;

		Log.i(TAG, "onStartCommand() end: " + mThread.isAlive() + ", " + mThread.getState());

		return START_REDELIVER_INTENT;
	}

	@Override
	public void onRevoke() {
		Log.i(TAG, "Got onRevoke()");
		stop();
	}

	@Override
	public void onDestroy() {
		Log.i(TAG, "Got onDestroy()");
		stop();
	}

	public static boolean isRunning() {  
		return isRunning;  
	}
	public static boolean isConnected() {  
		return isConnected;  
	}

	public void stop() {
		Log.i(TAG, "native clean up");

		if (mThread != null) {
			mThread.interrupt();
		}
		Stop();

		Log.i(TAG, "java clean up");
		try{
			if (mInterface != null) {
				mInterface.close();
				mInterface = null;
			}
			if (tunnel != null) {
				tunnel.close();
				tunnel = null;
			}
		} catch (Exception e) {
			Log.e(TAG, "clean Got " + e.toString());
		}
		isRunning = false;

		Log.i(TAG, "all clean up");
		stopSelf();
	}

	@Override
	public boolean handleMessage(Message message) {
		if (message != null) {
			Toast.makeText(this, message.what, Toast.LENGTH_SHORT).show();
		}
		return true;
	}

	@Override
	public synchronized void run() {
		try {
			Log.i(TAG, "Starting");

			isConnected = false;

			// If anything needs to be obtained using the network, get it now.
			// This greatly reduces the complexity of seamless handover, which
			// tries to recreate the tunnel without shutting down everything.
			// In this demo, all we need to know is the server address.
			InetSocketAddress server = new InetSocketAddress(mServerAddress, Integer.parseInt(mServerPort));

			// We try to create the tunnel for several times. The better way
			// is to work with ConnectivityManager, such as trying only when
			// the network is avaiable. Here we just use a counter to keep
			// things simple.
			for (int attempt = 0; attempt < 3; attempt++) {
				mHandler.sendEmptyMessage(R.string.connecting);

				// Reset the counter if we were connected.
				if (run(server)) {
					attempt = 0;
				}

				// Sleep for a while. This also checks if we got interrupted.
				Thread.sleep(2000);
			}
			Log.i(TAG, "Giving up");
		} catch (Exception e) {
			Log.e(TAG, "Got " + e.toString());
		} finally {
			try {
				mInterface.close();
			} catch (Exception e) {
				// ignore
			}
			mInterface = null;
			mParameters = null;

			mHandler.sendEmptyMessage(R.string.disconnected);
			Log.i(TAG, "Exiting");

			isConnected = false;
		}
	}

	private boolean run(InetSocketAddress server) throws Exception {
		boolean connected = false;
		try {
			tunnel = SocketChannel.open();

			// Protect the tunnel before connecting to avoid loopback.
			if (!protect(tunnel.socket())) {
				throw new IllegalStateException("Cannot protect the tunnel");
			}

			// Connect to the server.
			tunnel.connect(server);
			tunnel.configureBlocking(false);

			tunnelFd = ParcelFileDescriptor.fromSocket(tunnel.socket());
			SetSocketFD(tunnelFd.getFd());
//			SetSocketFD(tunnelFd.detachFd());

			// Authenticate and configure the virtual network interface.
			String param = Handshake(mSharedSecret).trim();
			if (!param.equals("")){
				configure(param);

				mHandler.sendEmptyMessage(R.string.connected);
				connected = true;
				isConnected = true;

//				SetTunFD(mInterface.detachFd());
				SetTunFD(mInterface.getFd());

				Log.i(TAG, "native Loop() start");
				Loop();
				Log.i(TAG, "native Loop() end");
			}
			connected = false;
			throw new IllegalStateException("Timed out");

		} catch (InterruptedException e) {
			throw e;
		} catch (Exception e) {
			Log.e(TAG, "tunnel Got " + e.toString());
		} finally {
			try {
				tunnel.close();
			} catch (Exception e) {
				// ignore
			}
			Stop();
		}
		return connected;
	}

	private void configure(String parameters) throws Exception {
		// If the old interface has exactly the same parameters, use it!
		if (mInterface != null && parameters.equals(mParameters)) {
			Log.i(TAG, "Using the previous interface");
			return;
		}

		Log.i(TAG, "parameter: " + parameters);
		// Configure a builder while parsing the parameters.
		Builder builder = new Builder();
		for (String parameter : parameters.split(" ")) {
			String[] fields = parameter.split(",");
			try {
				switch (fields[0].charAt(0)) {
					case 'm':
						builder.setMtu(Short.parseShort(fields[1]));
						break;
					case 'a':
						builder.addAddress(fields[1], Integer.parseInt(fields[2]));
						break;
					case 'r':
						builder.addRoute(fields[1], Integer.parseInt(fields[2]));
						break;
					case 'd':
						builder.addDnsServer(fields[1]);
						break;
					case 's':
						builder.addSearchDomain(fields[1]);
						break;
				}
			} catch (Exception e) {
				throw new IllegalArgumentException("Bad parameter: " + parameter);
			}
		}

		// Close the old interface since the parameters have been changed.
		try {
			mInterface.close();
		} catch (Exception e) {
			// ignore
		}

		// Create a new interface using the builder and save the parameters.
		mInterface = builder.setSession(mServerAddress)
			.setConfigureIntent(mConfigureIntent)
			.establish();
		mParameters = parameters;
		Log.i(TAG, "New interface: " + parameters);
	}
}

