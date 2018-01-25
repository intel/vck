all: validate

validate: validate_stream_prediction validate_model_training validate_interactive validate_batch_prediction

validate_batch_prediction:
	# Positive tests
	jsonschema -i examples/batch-prediction-job-valid-1.json batch-prediction-job-spec.json

	# Negative tests
	(jsonschema -i examples/batch-prediction-job-invalid-1.json batch-prediction-job-spec.json 2> /dev/null ; test $$? -eq 1)

validate_interactive:
	# Positive tests
	jsonschema -i examples/interactive-session-valid-1.json interactive-session-spec.json

	# Negative tests
	(jsonschema -i examples/interactive-session-invalid-1.json interactive-session-spec.json 2> /dev/null ; test $$? -eq 1)

validate_model_training:
	# Positive tests
	jsonschema -i examples/model-training-job-valid-1.json model-training-job-spec.json

	# Negative tests
	(jsonschema -i examples/model-training-job-invalid-1.json model-training-job-spec.json 2> /dev/null ; test $$? -eq 1)

validate_stream_prediction:
	# Positive tests
	jsonschema -i examples/stream-prediction-job-valid-1.json stream-prediction-job-spec.json

	# Negative tests
	(jsonschema -i examples/stream-prediction-job-invalid-1.json stream-prediction-job-spec.json 2> /dev/null ; test $$? -eq 1)
