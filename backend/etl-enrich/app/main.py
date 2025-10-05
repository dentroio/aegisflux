"""Main entry point for the ETL enrichment service."""

import asyncio
import logging
import signal
import sys
from typing import Optional

from .config import config
from .consumer import EventConsumer, setup_signal_handlers

# Configure logging
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


async def main():
    """Main application entry point."""
    logger.info("Starting ETL enrichment service")
    logger.info(f"Configuration: env={config.AF_ENV}, fake_rdns={config.AF_FAKE_RDNS}")
    logger.info(f"NATS URL: {config.NATS_URL}")
    logger.info(f"TimescaleDB: {config.PG_HOST}:{config.PG_PORT}/{config.PG_DB}")
    logger.info(f"Neo4j: {config.NEO4J_URI}")
    
    consumer = None
    try:
        # Create consumer
        consumer = EventConsumer(max_inflight=100)
        
        # Setup signal handlers for graceful shutdown
        await setup_signal_handlers(consumer)
        
        # Connect to services
        await consumer.connect()
        
        # Start the consumer
        logger.info("Starting ETL consumer...")
        await consumer.start()
        
    except KeyboardInterrupt:
        logger.info("Received keyboard interrupt")
    except Exception as e:
        logger.error(f"Fatal error: {e}")
        sys.exit(1)
    finally:
        # Cleanup
        logger.info("Shutting down ETL enrichment service...")
        if consumer:
            await consumer.disconnect()
        logger.info("ETL enrichment service stopped")


if __name__ == "__main__":
    asyncio.run(main())

