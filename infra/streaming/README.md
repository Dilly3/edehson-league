# Phone broadcasting experiment

Independent Terraform root and S3 state key. Commands here do not manage the production app.
Creates one public-playback IVS Low-Latency BASIC channel and one secret stream key in eu-west-1 (Ireland). The state bucket remains in eu-west-2. Recording disabled. BASIC does not generate multiple playback qualities.

## Create

```sh
cd ~/Desktop/edehson-league/infra/streaming
terraform init -backend-config=backend.hcl
terraform plan -out=streaming.tfplan
terraform apply streaming.tfplan
rm streaming.tfplan
```

## Connect Larix on iPhone

Retrieve the secret connection URL locally (never commit, screenshot or share it):

```sh
terraform output -raw larix_connection_url
```

Larix: Settings → Connections → add a connection named Edehsons test. Paste the full URL into its URL field. Enable that connection. Allow camera/microphone. Start with landscape 1280×720, 30fps, H.264, 2Mbps video, AAC audio and a 2-second keyframe interval. Exact menus vary by app version. Use the rear camera and keep the app foregrounded.

In AWS console, open Amazon IVS → Low-latency streaming → Channels → edehsons-streaming-test. Start broadcasting in Larix and watch in the channel's live preview. This does not add a player to the league website.

Test for 10–15 minutes using the pitch's mobile network, then stop broadcasting in Larix. Input and viewer usage are chargeable. Playback is public to anyone with its URL; the stream key controls broadcasting. The sensitive output is hidden in normal Terraform output but remains in protected remote state and saved plans.

## Remove experiment

Stop broadcasting first, then:

```sh
cd ~/Desktop/edehson-league/infra/streaming
terraform plan -destroy
terraform destroy
```

Do not run destroy in the parent infra folder. Destroying the channel invalidates its stream key; Terraform state history remains in the versioned backend bucket.
