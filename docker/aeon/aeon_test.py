from aeon import DataLoader
import pytest
import argparse
import os
import shutil

def test_without_data(data_dir, cache_dir):
    shutil.rmtree(data_dir)
    test(data_dir, cache_dir)


def test_with_data(data_dir, cache_dir):
    test(data_dir, cache_dir)


def test(data_dir, cache_dir):
    manifest_file = os.path.join("train.csv")

    batch_size=1

    subset_fraction=1.0

    image_config = {"type": "image",
                    "height": 224,
                    "width": 224}

    label_config = {"type": "label",
                    "binary": True}

    if not os.path.exists(cache_dir):
        os.makedirs(cache_dir)
    
    manifest = {'manifest_filename': manifest_file,
                'manifest_root': data_dir,
                'batch_size': batch_size,
                'subset_fraction': subset_fraction,
                'block_size': 2,
                'cache_directory': cache_dir,
                'etl': [image_config, label_config]}

    loader = DataLoader(manifest)

    for i, (x, t) in enumerate(loader):
        assert x[1].shape == (1, 3, 224, 224)
        assert t[1].shape == (1, 1)
        assert [[1], [2]] in t[1]
    
    print("Tests ran successfully")

if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("-p", default="./images/", help="Default directory to find the manifest data")
    parser.add_argument("-c", default="/cache/", help="Cache directory to use")
    parser.add_argument("-d", default="true", help="Test with data")
    args = parser.parse_args()

    if str(args.d).lower() == "false":
        test_without_data(args.p, args.c)
    else:
        test_with_data(args.p, args.c)
        