package cs8425.vpn;

import android.app.Activity;
import android.os.Bundle;

import android.content.Intent;
import android.net.VpnService;
import android.util.Log;
import android.view.View;
import android.widget.TextView;
import android.widget.Button;
import android.widget.Toast;

public class MainActivity extends Activity implements View.OnClickListener {
	private TextView mServerAddress;
	private TextView mServerPort;
	private TextView mSharedSecret;
	private Button mConnect;
	private static Intent mIntent;
	private static VpnServiceNative mService = new VpnServiceNative();

	/** Called when the activity is first created. */
	@Override
	public void onCreate(Bundle savedInstanceState) {
		super.onCreate(savedInstanceState);
		setContentView(R.layout.main);

		mServerAddress = (TextView) findViewById(R.id.address);
		mServerPort = (TextView) findViewById(R.id.port);
		mSharedSecret = (TextView) findViewById(R.id.secret);

		mServerAddress.setText("your.server.address");
		mServerPort.setText("23456");
		mSharedSecret.setText("test123456");

		mConnect = (Button) findViewById(R.id.connect);
		mConnect.setOnClickListener(this);
	}

	@Override
	public void onClick(View v) {
		if(mService.isRunning()){
			Toast.makeText(this, "clicked stop VPN", Toast.LENGTH_SHORT).show();
			mService.onRevoke();
			mService.onDestroy();
			mConnect.setText(R.string.connect);
			mIntent = null;
		}else{
			Intent intent = VpnService.prepare(this);
			if (intent != null) {
				startActivityForResult(intent, 0);
			} else {
				onActivityResult(0, RESULT_OK, null);
			}
		}
	}

	@Override
	protected void onActivityResult(int request, int result, Intent data) {
		if (result == RESULT_OK) {
			String prefix = getPackageName();
			mIntent = new Intent(this, VpnServiceNative.class)
				.putExtra(prefix + ".ADDRESS", mServerAddress.getText().toString())
				.putExtra(prefix + ".PORT", mServerPort.getText().toString())
				.putExtra(prefix + ".SECRET", mSharedSecret.getText().toString());
			startService(mIntent);
			mConnect.setText("stop");
		}
	}

}
